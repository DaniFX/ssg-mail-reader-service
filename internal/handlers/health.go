package handlers

import (
	"net/http"

	"github.com/DaniFX/ssg-mail-reader-service/internal/models"
	"github.com/gin-gonic/gin"
)

// HealthCheck risponde con 200 OK quando il servizio è up and running
func HealthCheck(c *gin.Context) {
	// Restituiamo una semplice risposta di successo standardizzata
	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{
		"status":  "UP",
		"service": "mail-reader-service",
	}, nil))
}
