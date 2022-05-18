package worker

import (
	"context"
	"github.com/JacoobH/crontab/common"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

//用来创建etcd服务
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	G_jobMgr *JobMgr
)

//监听任务
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		jobKey     string
		getResp    *clientv3.GetResponse
		kvPair     *mvccpb.KeyValue
		job        *common.Job
		revision   int64
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName    string
		jobEvent   *common.JobEvent
	)
	jobKey = common.JOB_SAVE_DIR

	//1 初始化，将已有任务先推送到scheduler调度协程
	//1.1先获取到所有的任务
	if getResp, err = jobMgr.kv.Get(context.TODO(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}
	//1.2将获取到的所有任务推送到scheduler(调度协程)
	for _, kvPair = range getResp.Kvs {
		//需要转json
		if job, err = common.UnpackJob(kvPair.Value); err != nil {
			//无法序列化的就跳过
			err = nil
			continue
		}
		jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
		//fmt.Println(*jobEvent)
		//把这个job同步给scheduler(调度协程)
		G_scheduler.PushJobEvent(jobEvent)
	}
	//2 打开一个协程，用来监听job任务变化，再将变化推送到scheduler调度协程
	go func() {
		//2.1 获取当前版本号
		revision = getResp.Header.Revision + 1
		//2.2开始监听(/cron/jobs/)
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(revision), clientv3.WithPrefix())
		//watchResp里有切片，无法直接比较空，只能用for...range ,不能用select
		for watchResp = range watchChan {
			//因为etcd在watch的时候，为了高吞吐量以及效率，所以有可能一次会发送多种监听事件在event中
			//然后存入chan
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:
					//保存任务，需要修改后的整体job
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						//有错误就跳过
						err = nil
						continue
					}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:
					//删除任务，所以只要key就好
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, &common.Job{
						Name: jobName,
					})
				}
				//将修改任务推入到scheduler调度协程
				//将删除任务推入到scheduler调度协程
				G_scheduler.PushJobEvent(jobEvent)
				//fmt.Println(*jobEvent)
			}
		}
	}()

	return
}

func (jobMgr *JobMgr) watchKiller() {
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
	)
	go func() {
		//开始监听(/cron/killer/)
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		//watchResp里有切片，无法直接比较空，只能用for...range ,不能用select
		for watchResp = range watchChan {
			//因为etcd在watch的时候，为了高吞吐量以及效率，所以有可能一次会发送多种监听事件在event中
			//然后存入chan
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // kill job event
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, &common.Job{
						Name: jobName,
					})
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: //kill mark expired, auto delete
				}
			}
		}
	}()
}

func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	//通过配置文件来读取
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}
	//启动监听新增编辑删除任务
	G_jobMgr.watchJobs()

	//启动监听强杀任务
	G_jobMgr.watchKiller()

	return
}

func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)
	return
}
