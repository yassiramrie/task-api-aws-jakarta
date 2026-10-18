package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTaskLifecycle(t *testing.T) {
	h := newAPI().routes()
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"review deployment"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", create.Code, create.Body.String())
	}
	var item task
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID != 1 || item.Title != "review deployment" {
		t.Fatalf("unexpected task: %+v", item)
	}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/tasks/1", http.StatusOK},
		{http.MethodDelete, "/tasks/1", http.StatusNoContent},
		{http.MethodGet, "/tasks/1", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.want {
			t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, w.Code, tc.want)
		}
	}
}

func TestRejectInvalidTask(t *testing.T) {
	h := newAPI().routes()
	for _, body := range []string{`{"title":""}`, `{"title":"ok","admin":true}`, `{`, `{"title":"ok"}{"title":"extra"}`} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %q: got %d", body, w.Code)
		}
	}
}
