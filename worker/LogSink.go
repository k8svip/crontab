package worker

import (
	"context"
	"github.com/k8svip/crontab/common"
	"github.com/mongodb/mongo-go-driver/mongo"
	"time"
)

// mongodb存储日志
type LogSink struct {
	client         *mongo.Client
	logCollection  *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

// 批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	logSink.logCollection.InsertMany(context.TODO(), batch.Logs)
}

//日志存储协程
func (logSink *LogSink) writeLoop() {
	var (
		log         *common.JobLog
		logBatch    *common.LogBatch //当前的批次
		commitTimer *time.Timer
		timeoutBatch *common.LogBatch //超时批次
	)

	for {
		select {
		case log = <-logSink.logChan:
			// 每次插入需要等待mongodb的一次请求往返，耗时可以因为网络慢花费比较长的时间
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				// 让这个批次超时，自动提交（给1秒的时间）
				commitTimer = time.AfterFunc(
					time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch),
				)
			}
			// 把新的日志追回到批次中
			logBatch.Logs = append(logBatch.Logs, log)

			//如果批次滿了，就立即发送
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				//发送日志
				logSink.saveLogs(logBatch)
				//清空batch
				logBatch = nil

				//取消定时器
				commitTimer.Stop()
			}
		case timeoutBatch = <- logSink.autoCommitChan: //过期的批次
			//判断过期批次是否仍个是当前的批次
			if timeoutBatch != logBatch{
				continue //跳过已经提交的批次
			}
			// 把批次写入到mongodb
			logSink.saveLogs(timeoutBatch)
			//清空batch
			logBatch = nil
		}
	}
}

func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(), G_config.MongodbUri); err != nil {
		return
	}

	// 选择DB
	G_logSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}

	// 启动一个mongodb处理协程
	go G_logSink.writeLoop()

	return
}

// 发送日志

func (logSink *LogSink) Append(jobLog *common.JobLog)  {

	select {
	case logSink.logChan <- jobLog:
	default:
		//队列满了就丢弃
	}
	//logSink.logChan <- jobLog
}