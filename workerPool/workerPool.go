package workerpool

import (
	"IO_BOUND/storage"
	"IO_BOUND/task"
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type TaskHandler interface {
	Type() string
	Handle(ctx context.Context, t *task.Task) error
}

type WorkerPool struct {
	db           *storage.Storage       // хранилище задач
	handlers     map[string]TaskHandler // оброботчики для определенных типов задач
	taskQueue    chan *task.Task        // очередь из задач
	totalWorkers int                    // количество воркеров
}

func New(db *storage.Storage, totalWorkers int) *WorkerPool {
	return &WorkerPool{
		db:           db,
		handlers:     make(map[string]TaskHandler),
		taskQueue:    make(chan *task.Task, 100),
		totalWorkers: totalWorkers,
	}
}

func (wp *WorkerPool) Add(t *task.Task) {
	wp.taskQueue <- t
}

func (wp *WorkerPool) NewHandler(handler TaskHandler) {
	wp.handlers[handler.Type()] = handler
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for range wp.totalWorkers {
		go wp.worker(ctx)
	}
}

func (wp *WorkerPool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-wp.taskQueue:
			wp.runTask(ctx, t)
		}
	}
}

func (wp *WorkerPool) runTask(ctx context.Context, t *task.Task) {
	// 1. Проверяем, не выполняется ли эта задача уже
	if data, err := wp.db.Get(ctx, t.GetKey()); err == nil {
		if len(data) == 0 {
			log.Printf("data with len = 0")
			return
		}
		existTask := task.Task{}
		err = json.Unmarshal(data, &existTask)
		if err != nil {
			log.Printf("Failed to unmarshal: %v", err)
			return
		}
		if existTask.Status != 0 {
			return // Задача занята
		}
	}
	// 2. Обрабатываем задачу
	t.Status = task.Runing
	if err := wp.db.Save(ctx, t.GetKey(), t); err != nil {
		log.Printf("Failed to update task status: %v", err)
		return
	}
	TH, exist := wp.handlers[t.Type]
	if !exist {
		t.Status = task.Failed
		t.Error = fmt.Errorf("no handler registred")
		err := wp.db.Save(ctx, t.GetKey(), t)
		if err != nil {
			log.Printf("Failed to update task status: %v", err)
		}
		return
	}
	// 3. Запускаем задачу
	err := TH.Handle(ctx, t)
	if err != nil {
		t.Status = task.Failed
		t.Error = fmt.Errorf("error while runing task: %w", err)
		err := wp.db.Save(ctx, t.GetKey(), t)
		if err != nil {
			log.Printf("Failed to update task status: %v", err)
		}
		return
	}
	err = wp.db.Save(ctx, t.GetKey(), t)
	if err != nil {
		log.Printf("Failed to update task status: %v", err)
	}
}
