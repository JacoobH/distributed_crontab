package worker

import (
	"context"
	"github.com/JacoobH/crontab/common"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// JobMgr Used to create the ETCD service
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	G_jobMgr *JobMgr
)

//Listening jobs
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

	//1 Initialization to push existing jobs to scheduler scheduling coroutine first
	//1.1 Get all quests first
	if getResp, err = jobMgr.kv.Get(context.TODO(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}
	//1.2 Push all jobs fetched to the Scheduler (scheduled coroutine)
	for _, kvPair = range getResp.Kvs {
		//Need to turn to json
		if job, err = common.UnpackJob(kvPair.Value); err != nil {
			//Those that cannot be serialized are skipped
			err = nil
			continue
		}
		jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
		//fmt.Println(*jobEvent)
		//Synchronize this job to the scheduler
		G_scheduler.PushJobEvent(jobEvent)
	}
	//2 Open a coroutine that monitors job changes and pushes the changes to scheduler
	go func() {
		//2.1 Gets the current version number
		revision = getResp.Header.Revision + 1
		//2.2 Start listening (/cron/jobs/)
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(revision), clientv3.WithPrefix())
		//WatchResp has a slice in it，you can't compare empty directly，can only be used for... Range, you can't use select
		for watchResp = range watchChan {
			//For high throughput and efficiency, etcd may send multiple listening events in event at one time when it is on watch
			//And then put it in Chan
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:
					//Save the job that needs to be modified
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						//Skip any mistakes
						err = nil
						continue
					}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:
					//Delete the job, so just need the key
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, &common.Job{
						Name: jobName,
					})
				}
				//Push the change job into the Scheduler scheduling coroutine
				//The delete-job is pushed into the Scheduler scheduling coroutine
				G_scheduler.PushJobEvent(jobEvent)
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
