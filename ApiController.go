package main

import (
	"github.com/gin-gonic/gin"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"io"
	"net/http"
	"strconv"
)

type ApiController struct {
	out     io.Writer
	storage StorageInterface
}

func (controller *ApiController) getStudentDisciplines(c *gin.Context) {
	studentId, _ := strconv.Atoi(c.Param("student_id"))
	if studentId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect student_id: " + c.Param("student_id"),
		})

	} else {
		disciplineScoreResults, err := controller.storage.getDisciplineScoreResultsByStudentId(studentId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, scoreApi.ErrorResponse{
				Error: err.Error(),
			})

		} else {
			c.JSON(http.StatusOK, disciplineScoreResults)
		}
	}
}

func (controller *ApiController) getStudentDiscipline(c *gin.Context) {
	studentId, _ := strconv.Atoi(c.Param("student_id"))
	disciplineId, _ := strconv.Atoi(c.Param("discipline_id"))

	if studentId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect student_id: " + c.Param("student_id"),
		})
	} else if disciplineId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect discipline_Id: " + c.Param("discipline_id"),
		})

	} else {
		disciplineScoreResult, err := controller.storage.getDisciplineScoreResultByStudentId(studentId, disciplineId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, scoreApi.ErrorResponse{
				Error: err.Error(),
			})

		} else if disciplineScoreResult.Discipline.Id == 0 {
			c.JSON(http.StatusNotFound, scoreApi.ErrorResponse{
				Error: "Discipline not exists: " + c.Param("discipline_id"),
			})

		} else {
			c.JSON(http.StatusOK, disciplineScoreResult)
		}
	}
}

func (controller *ApiController) getStudentDisciplineScore(c *gin.Context) {
	studentId, _ := strconv.Atoi(c.Param("student_id"))
	disciplineId, _ := strconv.Atoi(c.Param("discipline_id"))
	lessonId, _ := strconv.Atoi(c.Param("lesson_id"))

	if studentId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect student_id: " + c.Param("student_id"),
		})
	} else if disciplineId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect discipline_Id: " + c.Param("discipline_id"),
		})
	} else if lessonId <= 0 {
		c.JSON(http.StatusBadRequest, scoreApi.ErrorResponse{
			Error: "Incorrect lesson_id: " + c.Param("lesson_id"),
		})
	} else {
		disciplineScore, err := controller.storage.getDisciplineScore(studentId, disciplineId, lessonId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, scoreApi.ErrorResponse{
				Error: err.Error(),
			})

		} else if disciplineScore.Discipline.Id == 0 {
			c.JSON(http.StatusNotFound, scoreApi.ErrorResponse{
				Error: "Discipline not exists: " + c.Param("discipline_id"),
			})

		} else if disciplineScore.Score.Lesson.Id == 0 {
			c.JSON(http.StatusNotFound, scoreApi.ErrorResponse{
				Error: "Lesson not exists: " + c.Param("lesson_id"),
			})

		} else {
			c.JSON(http.StatusOK, disciplineScore)
		}
	}
}
