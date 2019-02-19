package master

import (
	"context"
	"fmt"
	"github.com/k8svip/crontab/common"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
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
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(), G_config.MongodbUri); err != nil {
		return
	}

	G_logMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}
	return
}

// 查看任务日志
func (logMgr *LogMgr) ListLog(name *common.JobLog, skip, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter  *common.JobLogFilter
		logSort *common.SortLogByStartTime
		cursor  mongo.Cursor
		jobLog *common.JobLog

	)

	// len(logArr)
	//logArr = make([] *common.JobLog,0)

	// 过滤条件
	//filter = &common.JobLogFilter{JobName: name}
	filter = &common.JobLogFilter{JobName:name.JobName,JobGroup:name.JobGroup}
	fmt.Println("1:",filter.JobName,"2:",filter.JobGroup)

	// 按照任务时间开始倒排；
	logSort = &common.SortLogByStartTime{SortOrder: -1}

	// 查询
	if cursor, err = logMgr.logCollection.Find(context.TODO(), filter, options.Find().SetLimit(int64(limit)), options.Find().SetSkip(int64(skip)), options.Find().SetSort(logSort)); err != nil {
		return
	}
	defer cursor.Close(context.TODO())

	// 遍历
	for cursor.Next(context.TODO()){
		jobLog = &common.JobLog{}
		// 反序列化bason
		if err = cursor.Decode(jobLog);err != nil {
			continue //有日志不合法
		}

		logArr = append(logArr, jobLog)
	}
	return
}
