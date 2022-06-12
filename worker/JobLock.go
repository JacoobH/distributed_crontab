package worker

import (
	"context"
	"github.com/JacoobH/crontab/common"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type JobLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	JobName    string
	cancelFunc context.CancelFunc
	leaseId    clientv3.LeaseID
	isLocked   bool
}

func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		JobName: jobName,
	}
	return
}

// TryLock Try to get lock
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResponse *clientv3.LeaseGrantResponse
		cancelCtx          context.Context
		cancelFunc         context.CancelFunc
		leaseId            clientv3.LeaseID
		keepRespChan       <-chan *clientv3.LeaseKeepAliveResponse
		txn                clientv3.Txn
		lockKey            string
		txnResp            *clientv3.TxnResponse
	)
	// 1.Create lease(5s)
	if leaseGrantResponse, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	// Used to cancel automatic renewal
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())

	leaseId = leaseGrantResponse.ID
	// 2.Automatic renewal
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FAIL
	}
	// 3.Coroutines that handle lease renewal replies
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan:
				if keepResp == nil {
					goto END
				}
			}
		}
	END:
	}()
	// 4.Create transaction txn
	txn = jobLock.kv.Txn(context.TODO())

	lockKey = common.JOB_LOCK_DIR + jobLock.JobName
	// 5.Transaction grab the lock
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	// commit transaction
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}
	// 6.If success then return, else release the lease
	if !txnResp.Succeeded {
		err = common.ERR_CLOCK_ALREADY_REQUIRED
		goto FAIL
	}

	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return
FAIL:
	cancelFunc()
	jobLock.lease.Revoke(context.TODO(), leaseId)
	return
}

func (jobLock *JobLock) Unlock() {
	//Cancel the coroutine for automatic lease renewal
	if jobLock.isLocked {
		jobLock.cancelFunc()
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId)
	}
}
