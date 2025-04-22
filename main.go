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

type PdfHandler struct {
	t string
}

func (pdf *PdfHandler) Type() string {
	return pdf.t
}

func (pdf *PdfHandler) Handle(ctx context.Context, t *task.Task) error {
	log.Printf("task with ID: %s started", t.ID)
	time.Sleep(1 * time.Minute)
	log.Printf("task with ID: %s finished", t.ID)
	return nil
}

func main() {
	ctx := context.Background()
	strg, err := storage.New(ctx, "localhost:6380", "ADMIN", "USER_PASSWORD")
	if err != nil {
		log.Fatalln(err)
	}
	// strg.Delete(ctx, "pdfHandler:test content")
	// strg.GetAll(ctx)
	// // strg.Save(ctx, t.GetKey(), &t)
	// data, err := strg.Get(ctx, t.GetKey())
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(string(data))
	wp := workerpool.New(strg, 5)
	wp.NewHandler(&PdfHandler{"pdfHandler"})
	wp.Start(ctx)
	// for i := 0; i < 10; i++ {
	// 	// t := task.Task{
	// 	// 	ID:       uuid.New().String(),
	// 	// 	Type:     "pdfHandler",
	// 	// 	Status:   task.Pending,
	// 	// 	Input:    strconv.Itoa(i),
	// 	// 	CreateAt: time.Now(),
	// 	// 	UpdateAt: time.Now(),
	// 	// }
	// 	// wp.Add(&t)
	// strg.Delete(ctx, fmt.Sprintf("pdfHandler:%s", "pdfHandler:test content"))
	// }
	// wp.Stop()
	// str, _ := strg.Get(ctx, "pdfHandler:test content")
	// fmt.Println(string(str))
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
	// Запуск сервера
	log.Println("Starting server on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
