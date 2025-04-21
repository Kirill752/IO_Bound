package server

import (
	"IO_BOUND/task"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, key string, t *task.Task) error
	Get(ctx context.Context, id string) (*task.Task, error)
}

type WorkerPool interface {
	Add(t *task.Task)
}

type Server struct {
	Rep Repository
	WP  WorkerPool
}

func New(repo Repository, WP WorkerPool) *Server {
	return &Server{
		Rep: repo,
		WP:  WP,
	}
}

func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Обработка запроса
	var req task.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	// Создание задачи с уникальным ID
	newTask := &task.Task{
		ID:       uuid.New().String(),
		Type:     req.Type,
		Status:   task.Pending,
		Input:    req.Text,
		CreateAt: time.Now(),
		UpdateAt: time.Now(),
	}
	// Сохранение задачи в базе
	if err := s.Rep.Save(r.Context(), newTask.GetKey(), newTask); err != nil {
		http.Error(w, "unable to create task", http.StatusInternalServerError)
		return
	}
	// Добавляем задачу в очередь
	s.WP.Add(newTask)
	// Возвращаем ответ пользователю, о состоянии задачи
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":     newTask.ID,
		"status": newTask.GetStatus(),
	})
}

func (s *Server) GetTask(w http.ResponseWriter, r *http.Request) {
	// Считываем ID
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	// Получаем задачу
	res, err := s.Rep.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	// Возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
