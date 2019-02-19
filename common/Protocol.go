package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"net"
	"strings"
	"time"
)

// 定时任务
type Job struct {
	Name      string `json:"name"`      // 任务名
	Command   string `json:"command"`   // shll命令
	CronExpr  string `json:"cronExpr"`  // cron 表达式
	JobStatus int    `json:"jobStatus"` // 任务是否开启
	JobGroup  string `json:"jobGroup"`  // job所属组
}

type GroupInfo struct {
	GroupName string `json:"groupName"` // 组名
	GroupIps  string `json:"groupIps"`  // 组IP信息
}

// 任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression //解析好的cronexpr表达式
	NextTime time.Time            //任务的下次执行时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job        *Job               // 任务信息
	PlanTime   time.Time          // 理论上的调度时间
	RealTime   time.Time          // 实际的调度时间
	CancelCtx  context.Context    //用于取消任务的context
	CancelFunc context.CancelFunc //用于取消command执行的cancel函数
}

// HTTP接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // SAVE ,DELETE事件
	Job       *Job
}

// 任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo // 执行状态
	Output      []byte          // 脚本的输出；
	Err         error           // 脚本错误原因
	StartTime   time.Time       // 启动时间
	EndTime     time.Time       // 结束时间
}

// 任务执行日志
type JobLog struct {
	JobGroup       string `json:"jobGroup" bson:"jobGroup"`
	JobExecutingIP string `json:"jobExecutingIP" bson:"jobExecutingIP"`
	JobName        string `json:"jobName" bson:"jobName"`           // 任务名
	Command        string `json:"command" bson:"command"`           // 脚本命令
	Err            string `json:"err" bson:"err"`                   // 错误原因
	Output         string `json:"output" bson:"output"`             //脚本输出
	PlanTime       int64  `json:"planTime" bson:"planTime"`         //计划开始时间
	ScheduleTime   int64  `json:"scheduleTime" bson:"scheduleTime"` //任务执行开始时间
	StartTime      int64  `json:"startTime" bson:"startTime"`       //任务开始时间
	EndTime        int64  `json:"endTime" bson:"endTime"`           //任务执行结束时间
}

// 日志批次
type LogBatch struct {
	Logs []interface{} //多条日志
}

// 任务日志过滤条件
type JobLogFilter struct {
	JobGroup string `bson:"jobGroup"`
	JobName  string `bson:"jobName"`
}

// 任务日志排序条件
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` // 排序条件是倒序排列{startTime:-1}
}

// 应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	//1. 定义一个response
	var (
		response Response
	)

	response.Errno = errno
	response.Msg = msg
	response.Data = data

	// 2. 序列化json

	resp, err = json.Marshal(response)

	return
}

// 反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		return
	}
	ret = job
	return
}

// 反序列化Job
func UnpackGroupInfo(value []byte) (ret *GroupInfo, err error) {
	var (
		groupInfo *GroupInfo
	)
	groupInfo = &GroupInfo{}
	if err = json.Unmarshal(value, groupInfo); err != nil {
		return
	}
	ret = groupInfo
	return
}

// 从etcd的key中提取任务名
// /cron/jobs/job10 抹掉/cron/jobs/
func ExtractJobName(jobKey string) (string) {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// /cron/killer/job10 抹掉/cron/killer/
func ExtractKillerName(killerKey string) (string) {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

// 任务变化事件有2种，1. 更新任务，2. 删除任务
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// 构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	//首先解析JOb的cron表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	//生成任务调度计划对象
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	return
}

// 构造执行状态信息
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime, //计算调度时间
		RealTime: time.Now(),               // 真实调度时间
	}

	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())

	return
}

// 提取worker的IP
func ExtractWorkerIP(regKey string) (string) {
	return strings.TrimPrefix(regKey, JOB_WORKER_DIR)
}

func ExtractWorkerGroup(regKey string) (string) {
	return strings.TrimPrefix(regKey, JOB_WORKER_GROUP_DIR)
}

func KeyJoin(list []string, sep string) (string) {
	return strings.Join(list, sep)
}

func GetLocalIPCheck() (ipv4 string, err error) {
	// 获取所有网卡的
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)

	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP

	for _, addr = range addrs {
		// IPv4, ipv6
		// 这个网络地址是IP地址
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过ipv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // 11.1.1.1
				return
			}

		}
	}
	err = ERR_NO_LOCAL_IP_FOUND
	return
}
