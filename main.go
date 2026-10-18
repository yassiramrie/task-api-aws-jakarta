package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

type api struct {
	mu       sync.RWMutex
	tasks    map[int64]task
	nextID   int64
	requests atomic.Uint64
}

func newAPI() *api { return &api{tasks: make(map[int64]task)} }

func (a *api) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"service": "task-api", "tasks": "/tasks", "health": "/healthz"})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("# TYPE task_api_requests_total counter\ntask_api_requests_total " + strconv.FormatUint(a.requests.Load(), 10) + "\n"))
	})
	mux.HandleFunc("/tasks", a.tasksHandler)
	mux.HandleFunc("/tasks/", a.taskHandler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.requests.Add(1)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		mux.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (a *api) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		items := make([]task, 0, len(a.tasks))
		for _, item := range a.tasks {
			items = append(items, item)
		}
		a.mu.RUnlock()
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input struct {
			Title string `json:"title"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil || len(strings.TrimSpace(input.Title)) == 0 || len(input.Title) > 120 {
			http.Error(w, "title must contain 1 to 120 characters", http.StatusBadRequest)
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
			return
		}
		a.mu.Lock()
		a.nextID++
		item := task{ID: a.nextID, Title: strings.TrimSpace(input.Title), CreatedAt: time.Now().UTC()}
		a.tasks[item.ID] = item
		a.mu.Unlock()
		writeJSON(w, http.StatusCreated, item)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *api) taskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/tasks/"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		a.mu.RLock()
		item, ok := a.tasks[id]
		a.mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		a.mu.Lock()
		_, ok := a.tasks[id]
		delete(a.tasks, id)
		a.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: newAPI().routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("task API listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}
