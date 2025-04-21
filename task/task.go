package task

import (
	"fmt"
	"time"
)

const (
	Pending = iota
	Runing
	Done
	Failed
)

type Task struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Status   int       `json:"status"`
	Input    string    `json:"input"`
	Result   any       `json:"result,omitempty"`
	Error    error     `json:"error,omitempty"`
	CreateAt time.Time `json:"created_at"`
	UpdateAt time.Time `json:"updated_at"`
}

type TaskRequest struct {
	Type string
	Text string
}

func (t *Task) GetStatus() string {
	switch t.Status {
	case 0:
		return "pending"
	case 1:
		return "runing"
	case 2:
		return "done"
	case 3:
		return "failed"
	}
	return "unknown status"
}
func (t *Task) GetKey() string {
	return fmt.Sprintf("%s:%s", t.Type, t.Input)
}
