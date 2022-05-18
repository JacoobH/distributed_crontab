package worker

import (
	"context"
	"fmt"
	"github.com/JacoobH/crontab/common"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/mongo"
)

type LogSink struct {
	client         *mongo.Client
	collection     *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	//因为是插入日志，所以成功失败皆可
	logSink.collection.InsertMany(context.TODO(), batch.Logs)
}
func (logSink *LogSink) writeLoop() {
	var (
		log          *common.JobLog
		batch        *common.LogBatch //当前的日志批次
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch //超时批次
	)
	for {
		select {
		case log = <-logSink.logChan:
			//将获得到的日志进行写入
			if batch == nil {
				//证明是已经提交了或者是刚刚进入
				batch = &common.LogBatch{}
				//超过规定阈值，进行自动提交
				commitTimer = time.AfterFunc(time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond, func(batch *common.LogBatch) func() {
					//这里传入的batch会与外部的batch不相同
					return func() {
						//将传入的超时批次放到autoCommitChan，让select检索到去做后面的事情
						logSink.autoCommitChan <- batch
					}
				}(batch))

			}
			batch.Logs = append(batch.Logs, log)
			//查看目前数量是否达到阈值，达到就提交
			if len(batch.Logs) >= G_config.JobLogBatchSize {
				//提交
				logSink.saveLogs(batch)
				//清空
				batch = nil
				//已经自动提交了，就将定时器停止(取消)
				commitTimer.Stop()
			}
		case timeoutBatch = <-logSink.autoCommitChan:
			//超时批次
			//因为有可能刚发过来，日志马上满了。已经提交过了,batch就变化了
			if timeoutBatch != batch {
				continue //跳过提交
			}
			//提交
			logSink.saveLogs(timeoutBatch)
			//清空
			batch = nil
		}
	}
}

func InitLogSink() (err error) {
	var (
		client     *mongo.Client
		clientOps  *options.ClientOptions
		collection *mongo.Collection
	)
	clientOps = options.Client().
		ApplyURI(G_config.MongodbUri).
		SetConnectTimeout(time.Duration(G_config.MongodbConnectTimeout) * time.Millisecond)
	if client, err = mongo.Connect(context.TODO(), clientOps); err != nil {
		fmt.Println(err)
		return
	}
	//select table my_collection
	collection = client.Database("cron").Collection("log")

	G_logSink = &LogSink{
		client:         client,
		collection:     collection,
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}
	go G_logSink.writeLoop()
	return
}

func (logSink *LogSink) append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:
		//When the queue is full, discarded
	}
}
