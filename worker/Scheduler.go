package worker

import (
	"fmt"
	"github.com/JacoobH/crontab/common"
	"time"
)

// Scheduler job scheduling
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              // Etcd job event queue
	jobPlanTable      map[string]*common.JobSchedulePlan // Job scheduling table
	jobExecutingTable map[string]*common.JobExecuteInfo  // Job Execute table
	jobResultChan     chan *common.JobExecuteResult
}

var (
	G_scheduler *Scheduler
)

// Processing job Events
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExecuteInfo  *common.JobExecuteInfo
		jobExecuting    bool
		jobExisted      bool
		err             error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		// cancel command exec
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			fmt.Println("cancel job:", jobExecuteInfo.Job.Name)
			jobExecuteInfo.CancelFunc()
		}
	}
}

// TryStartJob Try to start job
func (scheduler *Scheduler) TryStartJob(jobSchedulePlan *common.JobSchedulePlan) {
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)

	// if there is a job executing, skip
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobSchedulePlan.Job.Name]; jobExecuting {
		fmt.Println("Not yet out, skip", jobSchedulePlan.Job.Name)
		return
	}

	// build execute status info
	jobExecuteInfo = common.BuildJobExecuteInfo(jobSchedulePlan)

	// save execute status info
	scheduler.jobExecutingTable[jobSchedulePlan.Job.Name] = jobExecuteInfo

	//TODO: exec job
	fmt.Println("exec job:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}

// TrySchedule Recalculate the task scheduling status
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)

	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}
	// current time
	now = time.Now()
	//1. Iterate through all jobs
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			scheduler.TryStartJob(jobPlan)
			fmt.Println("exec job:", jobPlan.Job.Name)
			jobPlan.NextTime = jobPlan.Expr.Next(now) // Updated the next execution time
		}
		// Count the last time a job expired
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	//Interval for next scheduling（earTime - now）
	scheduleAfter = (*nearTime).Sub(now)
	return
	//2. Expired jobs are executed immediately
	//3. Count the time of the most recent expired job (N seconds after expiration == scheduleAfter)
}

func (scheduler *Scheduler) handleJobResult(jobExecuteResult *common.JobExecuteResult) {
	var (
		jobLog *common.JobLog
	)
	// Deleting the Execution State
	delete(scheduler.jobExecutingTable, jobExecuteResult.JobExecuteInfo.Job.Name)

	// Generate execution Logs
	if jobExecuteResult.Err != common.ERR_CLOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      jobExecuteResult.JobExecuteInfo.Job.Name,
			Command:      jobExecuteResult.JobExecuteInfo.Job.Command,
			Output:       string(jobExecuteResult.OutPut),
			PlanTime:     jobExecuteResult.JobExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: jobExecuteResult.JobExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    jobExecuteResult.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      jobExecuteResult.EndTime.UnixNano() / 1000 / 1000,
		}
		if jobExecuteResult.Err != nil {
			jobLog.Err = jobExecuteResult.Err.Error()
		} else {
			jobLog.Err = ""
		}
		//TODO: save to mongodb
		G_logSink.append(jobLog)
	}
	fmt.Println("Task execution completed：", jobExecuteResult.JobExecuteInfo.Job.Name, jobExecuteResult.OutPut, jobExecuteResult.Err)
}

// scheduling coroutine
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)
	//Initialize(1sec)
	scheduleAfter = scheduler.TrySchedule()

	//Delay timer for scheduling
	scheduleTimer = time.NewTimer(scheduleAfter)
	// Timing job common.Job
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: // Listen for job change events
			//CRUD the job list maintained in memory
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: // The latest job is expired
		case jobResult = <-scheduler.jobResultChan: // Monitor the execution result of a task
			scheduler.handleJobResult(jobResult)
		}
		//scheduling job
		scheduleAfter = scheduler.TrySchedule()
		//Reset scheduling interval
		scheduleTimer.Reset(scheduleAfter)
	}
}

// PushJobEvent Pushing job change events
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

// InitScheduler Initialize scheduler
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}
	// Start scheduling coroutine
	go G_scheduler.scheduleLoop()
	return
}

func (scheduler *Scheduler) PushJobResult(jobExecuteResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobExecuteResult
}
