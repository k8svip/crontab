package worker

import (
	"fmt"
	"github.com/k8svip/crontab/common"
	"time"
)

//任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              //etcd任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  //执行状态表
	jobResultChan     chan *common.JobExecuteResult      // 任务结果队列
}

var (
	G_scheduler *Scheduler
)

// 处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExecuteInfo  *common.JobExecuteInfo
		jobExecuting    bool
		jobExisted      bool
		err             error
	)
	// 
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: //保存任务事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE: //删除任务事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL: // 强杀任务事件
		// 取消掉command执行；判断任务是否在执行中
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc() // 触发command杀死shell子进程，任务得到退出；
		}
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	var (
		jobExecuteInfo          *common.JobExecuteInfo
		jobExecuting            bool
		localIPCheckWithGroupIP string
		err                     error
		result                  bool
	)
	// 调度和执行是两件事情
	// 调度就是定期的检查哪些任务过期了，这就是调度，当我们发现一个过期任务时，之后要执行它，这叫执行；
	// 执行的任务有可能运行很久，有可能我们每秒执行一个任务，但这个任务要跑1分钟，哪么1分钟会调度60次，但是呢只能执行一次，因为这个任务一直没有退出，所以我们只能执行一次；防止并发，
	// 所以这里的函数名叫tryStartJob，尝试启动一个任务，所以我们需要记录一个状态，表时任务正在执行中，所以我们需要定义一个变量任务执行表；
	// 如果任务正在执行，跳过本次调度，

	// 判断任务状态，是否加入执行列表中；
	if jobPlan.Job.JobStatus == 0 {
		delete(scheduler.jobExecutingTable, jobPlan.Job.Name)
		return
	}
	if localIPCheckWithGroupIP, err = common.GetLocalIPCheck(); err != nil {
		return
	}

	fmt.Println(localIPCheckWithGroupIP)

	if result = G_checkGroupMgr.BaseGroupCheckLoclIP(jobPlan.Job.JobGroup, localIPCheckWithGroupIP); result == false {
		return
	}

	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		// fmt.Println("尚未退出，跳过此次执行，", jobPlan.Job.Name)
		return
	}

	// 构建执行状态；
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	// 保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务

	fmt.Println("执行任务：", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)

}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)
	// 如果任务表为空的话，随便睡眠多久
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	// 当前时间
	now = time.Now()
	// 遍历所有的任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			// TODO: 尝试执行任务
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) //更新下次执行时间
		}

		// 统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}
	// 下次调度间隔 (最近要执行的任务调度时间 - 当前时间)
	scheduleAfter = (*nearTime).Sub(now)

	return
}

// 处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	var (
		jobLog  *common.JobLog
		localIP string
		err     error
	)
	// 删除执行状态
	delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.Name)

	if localIP, err = common.GetLocalIPCheck(); err != nil {
		return
	}

	// 生成执行日志
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobGroup:       result.ExecuteInfo.Job.JobGroup,
			JobExecutingIP: localIP,
			JobName:        result.ExecuteInfo.Job.Name,
			Command:        result.ExecuteInfo.Job.Command,
			Output:         string(result.Output),
			PlanTime:       result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime:   result.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:      result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:        result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
		// TODO: 存储到mogodb;
		G_logSink.Append(jobLog)
	}
	fmt.Println("任务执行完成：", result.ExecuteInfo.Job.Name, string(result.Output), result.Err)
}

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)
	// 初始化一次，得到下次的执行时间；也里是1秒，因为len(scheduler.jobPlanTable)是0，还没有值；
	scheduleAfter = scheduler.TrySchedule()

	//调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	// 定时任务common.Job
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: // 监听任务变化事件
			// 对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: //最近的任务到期了
		case jobResult = <-scheduler.jobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}

		// 调度一次任务
		scheduleAfter = scheduler.TrySchedule()

		//重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}

}

// 推送任务变化事件
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}

	// 启动调度协程
	go G_scheduler.scheduleLoop()

	return
}

// 回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
