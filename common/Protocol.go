package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

// Job timed job
type Job struct {
	Name     string `json:"name" form:"name"`         // job name
	Command  string `json:"command" form:"command"`   // shell command
	CronExpr string `json:"cronExpr" form:"cronExpr"` // cron Expressions
}

// JobSchedulePlan Job scheduling plan
type JobSchedulePlan struct {
	Job      *Job                 // Information about the tasks to be scheduled
	Expr     *cronexpr.Expression // The resolved cronexpr expression
	NextTime time.Time            // Next scheduling time
}

// JobExecuteInfo Job exec status
type JobExecuteInfo struct {
	Job        *Job
	PlanTime   time.Time // theory
	RealTime   time.Time // real
	CancelCtx  context.Context
	CancelFunc context.CancelFunc
}

// Response Uniform return structure format
type Response struct {
	ErrNo int         `json:"errNo"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// JobEvent Change event
type JobEvent struct {
	EventType int //SAVE | DELETE
	Job       *Job
}

// JobLog Job log
type JobLog struct {
	JobName      string `bson:"jobName" json:"jobName"`
	Command      string `bson:"command" json:"command"`
	Err          string `bson:"err" json:"err"`
	Output       string `bson:"output" json:"output"`
	PlanTime     int64  `bson:"planTime" json:"planTime"`         //Planed start time
	ScheduleTime int64  `bson:"scheduleTime" json:"scheduleTime"` //Actual scheduling time
	StartTime    int64  `bson:"startTime" json:"startTime"`       //Start time of job execution
	EndTime      int64  `bson:"endTime" json:"endTime"`           //Job end time
}

// LogBatch Log batches
type LogBatch struct {
	Logs []interface{}
}

// JobExecuteResult result of job exec
type JobExecuteResult struct {
	JobExecuteInfo *JobExecuteInfo // exec status
	OutPut         []byte          // output
	Err            error           // error
	StartTime      time.Time       // start time
	EndTime        time.Time       // end time

}

// JobLogFilter Filtering criteria for task log query
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

// SortLogByStartTime Sort task log query
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"`
}

// BuildResponse Building the return format foundation
func BuildResponse(errNo int, msg string, data interface{}) (resp Response) {
	// 1. Define a response
	var (
		response Response
	)
	response.ErrNo = errNo
	response.Msg = msg
	response.Data = data

	// 2. Serialize json
	resp = response
	return
}

// UnpackJob Deserialize the job
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

// ExtractJobName Extract the job name from the JOB_SAVE_DIR directory of etcd
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// ExtractKillerName Extract the job name from the JOB_KILLER_DIR directory of etcd
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

// ExtractWorkerName Extract the job name from the JOB_WORKER_DIR directory of etcd
func ExtractWorkerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_WORKER_DIR)
}

// BuildJobEvent There are three types of task event changes: 1. Update a task 2. Delete a task 3. Terminate the task
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// BuildJobSchedulePlan Construct a job execution plan
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	// Parse the cron expression of the job
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {

	}

	// Generate JobSchedulePlan object
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

// BuildJobExecuteInfo Build a JobExecuteInfo
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime,
		RealTime: time.Now(),
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}
