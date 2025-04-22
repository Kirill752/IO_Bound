package server

import (
	"IO_BOUND/task"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	Save(ctx context.Context, key string, t *task.Task) error
	Get(ctx context.Context, id string) ([]byte, error)
	Close()
}

type WorkerPool interface {
	Add(t *task.Task)
	Stop()
}

type Server struct {
	Strg Storage
	WP   WorkerPool
}

func New(strg Storage, WP WorkerPool) *Server {
	return &Server{
		Strg: strg,
		WP:   WP,
	}
}

func (s *Server) SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// Прекращаем работу workerpool
			s.WP.Stop()
			log.Printf("task channel was closed")
			// Отсоединяемся от базы данных
			s.Strg.Close()
			os.Exit(0)
		}
	}()
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
	// Проверяем, есть ли такая задача в базе
	res, err := s.Strg.Get(r.Context(), newTask.GetKey())
	if err == nil {
		// Возвращаем ответ пользователю, о состоянии задачи
		w.Header().Set("Content-Type", "application/json")
		var out bytes.Buffer
		err = json.Indent(&out, res, "", "    ")
		if err != nil {
			http.Error(w, "error while json Indent", http.StatusInternalServerError)
		}
		w.Write(out.Bytes())
		return
	}
	// Сохранение задачи в базе
	if err := s.Strg.Save(r.Context(), newTask.GetKey(), newTask); err != nil {
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
	res, err := s.Strg.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	// Возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	var out bytes.Buffer
	err = json.Indent(&out, res, "", "    ")
	if err != nil {
		http.Error(w, "error while json Indent", http.StatusInternalServerError)
	}
	w.Write(out.Bytes())
}
