package worker

import (
	"github.com/JacoobH/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

// Executor Job executor
type Executor struct {
}

var (
	G_executor *Executor
)

// ExecuteJob Execute a job
func (executor *Executor) ExecuteJob(jobExecuteInfo *common.JobExecuteInfo) {
	go func() {
		var (
			cmd              *exec.Cmd
			err              error
			output           []byte
			jobExecuteResult *common.JobExecuteResult
			jobLock          *JobLock
		)
		jobExecuteResult = &common.JobExecuteResult{
			JobExecuteInfo: jobExecuteInfo,
			OutPut:         make([]byte, 0),
		}

		// Init lock
		jobLock = G_jobMgr.CreateJobLock(jobExecuteInfo.Job.Name)

		jobExecuteResult.StartTime = time.Now()

		//Sleep randomly for 0 ~ 1 seconds
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.Unlock()

		if err != nil {
			jobExecuteResult.Err = err
			jobExecuteResult.EndTime = time.Now()
		} else {
			// Reset the job startup time after capturing the lock（TryLock() is a network operation）
			jobExecuteResult.StartTime = time.Now()
			cmd = exec.CommandContext(jobExecuteInfo.CancelCtx, "/bin/bash", "-c", jobExecuteInfo.Job.Command)
			output, err = cmd.CombinedOutput()
			jobExecuteResult.EndTime = time.Now()
			jobExecuteResult.OutPut = output
			jobExecuteResult.Err = err
		}
		// When the job is completed, the result of the execution is returned to the Scheduler and deletes the execution record from the executingTable
		G_scheduler.PushJobResult(jobExecuteResult)
	}()
}

// InitExecutor Initializes the executor
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
