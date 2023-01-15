package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type ApiController struct {
	out     io.Writer
	storage StorageInterface
}

func (controller *ApiController) getStudentDisciplines(c *gin.Context) {

	c.String(http.StatusOK, "health")
}

func (controller *ApiController) getStudentDiscipline(c *gin.Context) {

	c.String(http.StatusOK, "health")
}
