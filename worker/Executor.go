package worker

import (
	"github.com/k8svip/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

// 任务执行器
type Executor struct {
}

var (
	G_executor *Executor
)

// 执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			jobLock *JobLock
		)
		//任务结果初始化
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		// 首先获取分布式锁，如果锁到了的话，就向下执行任务，如果锁不到，就跳过了，因为别的节点把它执行了，这个worker节点，就不需要执行了；
		// 初始化分布式锁
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)

		// 记录任务开始时间
		result.StartTime = time.Now()

		// 上锁
		//随机睡眠(0-1s)
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.UnLock()

		if err != nil { //上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {

			// 上锁成功后，重置任务启动时间
			result.StartTime = time.Now()

			// 执行shell命令
			cmd = exec.CommandContext(info.CancelCtx, "/bin/bash", "-c", info.Job.Command)

			// 执行并捕获输出
			output, err = cmd.CombinedOutput()

			// 记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err


		}
		// 把任务执行完成后，把执行的结果返回给Scheduler, Scheduler会从executingTable表中删除掉执行记录
		G_scheduler.PushJobResult(result)
	}()
}

// 初始化执行器

func InitExecutor() (err error) {
	G_executor = &Executor{

	}
	return
}
