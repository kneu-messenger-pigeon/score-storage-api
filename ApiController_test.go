package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetStudentDisciplines(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		out := &bytes.Buffer{}

		expectedResults := scoreApi.DisciplineScoreResults{
			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   100,
					Name: "Капітал!",
				},
				ScoreRating: scoreApi.ScoreRating{
					Total:         17,
					StudentsCount: 25,
					Rating:        8,
					MinTotal:      10,
					MaxTotal:      20,
				},
			},
			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   110,
					Name: "Гроші та лихварство",
				},
				ScoreRating: scoreApi.ScoreRating{
					Total:         12,
					StudentsCount: 25,
					Rating:        12,
					MinTotal:      7,
					MaxTotal:      17,
				},
			},
		}

		storage := NewMockStorageInterface(t)
		storage.On("getDisciplineScoreResultsByStudentId", 23).Return(expectedResults, nil)

		expectedBody, err := json.Marshal(expectedResults)
		assert.NoError(t, err)

		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/23/disciplines", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, expectedBody, w.Body.Bytes())
	})

	t.Run("Storage_error", func(t *testing.T) {
		out := &bytes.Buffer{}
		expectedError := errors.New("expected error")

		storage := NewMockStorageInterface(t)
		storage.On("getDisciplineScoreResultsByStudentId", 23).
			Return(scoreApi.DisciplineScoreResults{}, expectedError)

		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/23/disciplines", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, actualBody, "error")
	})

	t.Run("wrong student id ", func(t *testing.T) {
		out := &bytes.Buffer{}

		storage := NewMockStorageInterface(t)
		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/-99/disciplines", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, actualBody, "error")
	})
}

func TestGetStudentDiscipline(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		out := &bytes.Buffer{}
		expectedResult := scoreApi.DisciplineScoreResult{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			ScoreRating: scoreApi.ScoreRating{
				Total:         17,
				StudentsCount: 25,
				Rating:        8,
				MinTotal:      10,
				MaxTotal:      20,
			},
			Scores: []scoreApi.Score{
				{
					Lesson: scoreApi.Lesson{
						Id:   245,
						Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
						Type: scoreApi.LessonType{
							Id:        5,
							ShortName: "МК",
							LongName:  "Модульний контроль.",
						},
					},
					LessonHalf: 1,
					Score:      4.5,
					IsAbsent:   false,
				},
				{
					Lesson: scoreApi.Lesson{
						Id:   245,
						Date: time.Date(2023, time.Month(2), 14, 0, 0, 0, 0, time.Local),
						Type: scoreApi.LessonType{
							Id:        1,
							ShortName: "ПрЗн",
							LongName:  "Практичне зан.",
						},
					},
					LessonHalf: 2,
					Score:      0,
					IsAbsent:   true,
				},
			},
		}

		storage := NewMockStorageInterface(t)
		storage.On("getDisciplineScoreResultByStudentId", 23, 199).Return(expectedResult, nil)

		expectedBody, err := json.Marshal(expectedResult)
		assert.NoError(t, err)

		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/23/disciplines/199", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, expectedBody, w.Body.Bytes())
	})

	t.Run("not_exist_discipline", func(t *testing.T) {
		out := &bytes.Buffer{}
		expectedResult := scoreApi.DisciplineScoreResult{
			Discipline: scoreApi.Discipline{
				Id:   0,
				Name: "",
			},
		}

		storage := NewMockStorageInterface(t)
		storage.On("getDisciplineScoreResultByStudentId", 23, 199).Return(expectedResult, nil)

		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/23/disciplines/199", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, actualBody, "error")
	})

	t.Run("storage_error", func(t *testing.T) {
		out := &bytes.Buffer{}
		expectedError := errors.New("expected error")

		storage := NewMockStorageInterface(t)
		storage.On("getDisciplineScoreResultByStudentId", 23, 199).Return(scoreApi.DisciplineScoreResult{}, expectedError)

		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/23/disciplines/199", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, actualBody, "error")
	})

	t.Run("wrong student id ", func(t *testing.T) {
		out := &bytes.Buffer{}

		storage := NewMockStorageInterface(t)
		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/-99/disciplines/199", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, actualBody, "error")
	})

	t.Run("wrong discipline id ", func(t *testing.T) {
		out := &bytes.Buffer{}

		storage := NewMockStorageInterface(t)
		router := setupRouter(out, storage)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/v1/students/650/disciplines/0", nil)
		router.ServeHTTP(w, req)

		actualBody := gin.H{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, actualBody, "error")
	})
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
