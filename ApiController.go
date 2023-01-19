package main

import (
	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect student_id: " + c.Param("student_id")})

	} else {
		disciplineScoreResults, err := controller.storage.getDisciplineScoreResultsByStudentId(studentId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		} else {
			c.JSON(http.StatusOK, disciplineScoreResults)
		}
	}
}

func (controller *ApiController) getStudentDiscipline(c *gin.Context) {
	studentId, _ := strconv.Atoi(c.Param("student_id"))
	disciplineId, _ := strconv.Atoi(c.Param("discipline_id"))

	if studentId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect student_id: " + c.Param("student_id")})
	} else if disciplineId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect discipline_Id: " + c.Param("disciplineId")})

	} else {
		disciplineScoreResult, err := controller.storage.getDisciplineByStudentId(studentId, disciplineId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		} else if disciplineScoreResult.Discipline.Id == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Discipline not exists: " + c.Param("disciplineId")})

		} else {
			c.JSON(http.StatusOK, disciplineScoreResult)
		}
	}
}
