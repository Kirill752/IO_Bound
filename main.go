package main

import (
	"IO_BOUND/server"
	"IO_BOUND/storage"
	"IO_BOUND/task"
	workerpool "IO_BOUND/workerPool"
	"context"
	"log"
	"net/http"
	"time"
)

// Обработчик, который работает долгое время
type PdfHandler struct {
	t string
}

func (pdf *PdfHandler) Type() string {
	return pdf.t
}

func (pdf *PdfHandler) Handle(ctx context.Context, t *task.Task) error {
	log.Printf("task with ID: %s started", t.ID)
	time.Sleep(3 * time.Minute)
	log.Printf("task with ID: %s finished", t.ID)
	return nil
}

func main() {
	ctx := context.Background()
	strg, err := storage.New(ctx, "localhost:6380", "ADMIN", "USER_PASSWORD")
	if err != nil {
		log.Fatalln(err)
	}
	wp := workerpool.New(strg, 5)
	wp.NewHandler(&PdfHandler{"pdfHandler"})
	wp.Start(ctx)
	serv := server.New(strg, wp)
	serv.SetupGracefulShutdown()
	mux := http.NewServeMux()
	mux.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			serv.CreateTask(w, r)
		case http.MethodGet:
			serv.GetTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	log.Println("Starting server on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
