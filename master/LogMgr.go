package master

import (
	"context"
	"fmt"
	"github.com/JacoobH/crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	G_logMgr *LogMgr
)

func InitLogMgr() (err error) {
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

	G_logMgr = &LogMgr{
		client:        client,
		logCollection: collection,
	}
	return
}

func (logMgr *LogMgr) ListLog(name string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter  *common.JobLogFilter
		logSort *common.SortLogByStartTime
		findOpt *options.FindOptions
		cur     *mongo.Cursor
		jobLog  *common.JobLog
	)

	logArr = make([]*common.JobLog, 0)

	// Filter conditions
	filter = &common.JobLogFilter{JobName: name}
	logSort = &common.SortLogByStartTime{SortOrder: -1}

	findOpt = options.Find()
	findOpt.SetSort(logSort)
	findOpt.SetSkip(int64(skip))
	findOpt.SetLimit(int64(limit))

	logMgr.logCollection.Find(context.TODO(), filter, findOpt)

	if cur, err = logMgr.logCollection.Find(context.TODO(), filter, findOpt); err != nil {
		fmt.Println(err)
		return
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		if err = cur.Decode(jobLog); err != nil {
			continue
		}
		logArr = append(logArr, jobLog)
	}
	return
}
