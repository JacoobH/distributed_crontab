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
	//Because it is an insert log, success or failure is optional
	logSink.collection.InsertMany(context.TODO(), batch.Logs)
}
func (logSink *LogSink) writeLoop() {
	var (
		log          *common.JobLog
		batch        *common.LogBatch //The current log batch
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch //Timeout batches
	)
	for {
		select {
		case log = <-logSink.logChan:
			//Write to the obtained log
			if batch == nil {
				//Proof that it has been submitted or just entered
				batch = &common.LogBatch{}
				//If the threshold is exceeded, automatic submission is performed
				commitTimer = time.AfterFunc(time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond, func(batch *common.LogBatch) func() {
					//The batch passed in here will be different from the external batch
					return func() {
						//Place the passed timeout batch to the autoCommitChan and let the SELECT retrieve it to do the rest
						logSink.autoCommitChan <- batch
					}
				}(batch))

			}
			batch.Logs = append(batch.Logs, log)
			//Check whether the current quantity reaches the threshold, and submit when it does
			if len(batch.Logs) >= G_config.JobLogBatchSize {
				//Commit
				logSink.saveLogs(batch)
				//Clear
				batch = nil
				//If it has been submitted automatically, stop the timer (cancel)
				commitTimer.Stop()
			}
		case timeoutBatch = <-logSink.autoCommitChan:
			//Timeout batches
			//Because maybe it just came in and the log filled up. It has been submitted, and the batch has changed
			if timeoutBatch != batch {
				continue //Skip to submit
			}
			//Commit
			logSink.saveLogs(timeoutBatch)
			//Clear
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
