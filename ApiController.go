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
			Error: "Incorrect discipline_Id: " + c.Param("disciplineId"),
		})

	} else {
		disciplineScoreResult, err := controller.storage.getDisciplineScoreResultByStudentId(studentId, disciplineId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, scoreApi.ErrorResponse{
				Error: err.Error(),
			})

		} else if disciplineScoreResult.Discipline.Id == 0 {
			c.JSON(http.StatusNotFound, scoreApi.ErrorResponse{
				Error: "Discipline not exists: " + c.Param("disciplineId"),
			})

		} else {
			c.JSON(http.StatusOK, disciplineScoreResult)
		}
	}
}
