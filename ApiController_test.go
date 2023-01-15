package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStudentDisciplines(t *testing.T) {
	out := &bytes.Buffer{}

	storage := NewMockStorageInterface(t)

	router := setupRouter(out, storage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v1/students/1/disciplines", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "health", w.Body.String())
}

func TestGetStudentDiscipline(t *testing.T) {
	out := &bytes.Buffer{}

	storage := NewMockStorageInterface(t)

	router := setupRouter(out, storage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v1/students/1/disciplines/23", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "health", w.Body.String())
}

func TestPingRoute(t *testing.T) {
	out := &bytes.Buffer{}

	storage := NewMockStorageInterface(t)

	router := setupRouter(out, storage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "health", w.Body.String())
}
