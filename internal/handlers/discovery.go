package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetDiscovery restituisce il contratto del servizio per l'API Gateway
func GetDiscovery(c *gin.Context) {
	discoveryDoc := gin.H{
		"serviceName": "mail-reader-service",
		"version":     "1.0.0",
		"metadata": gin.H{
			"description": "Microservizio per la lettura e gestione delle email via protocollo IMAP",
		},
		"endpoints": []gin.H{
			{
				"path":         "/api/v1/mail/folders",
				"method":       "GET",
				"summary":      "Recupera la lista delle cartelle IMAP",
				"authRequired": true,
			},
			{
				"path":         "/api/v1/mail/folders",
				"method":       "POST",
				"summary":      "Crea una nuova cartella IMAP",
				"authRequired": true,
				"inputSchema": gin.H{
					"type":       "object",
					"properties": gin.H{"name": gin.H{"type": "string"}},
					"required":   []string{"name"},
				},
			},
			{
				"path":         "/api/v1/mail/search",
				"method":       "POST",
				"summary":      "Cerca email nel server IMAP in base a filtri",
				"authRequired": true,
				"inputSchema": gin.H{
					"type": "object",
					"properties": gin.H{
						"folder":       gin.H{"type": "string", "default": "INBOX"},
						"subject":      gin.H{"type": "string"},
						"from":         gin.H{"type": "string"},
						"bodyContains": gin.H{"type": "string"},
					},
				},
			},
			{
				"path":         "/api/v1/mail/messages/:uid",
				"method":       "GET",
				"summary":      "Recupera i dettagli e il corpo di una singola email tramite UID",
				"authRequired": true,
			},
			{
				"path":         "/api/v1/mail/messages/:uid/move",
				"method":       "PUT",
				"summary":      "Sposta un'email in una cartella di destinazione",
				"authRequired": true,
				"inputSchema": gin.H{
					"type":       "object",
					"properties": gin.H{"destinationFolder": gin.H{"type": "string"}},
					"required":   []string{"destinationFolder"},
				},
			},
			{
				"path":         "/api/v1/mail/messages/:uid",
				"method":       "DELETE",
				"summary":      "Elimina un'email (aggiunge flag \\Deleted o sposta nel cestino)",
				"authRequired": true,
			},
		},
	}

	c.JSON(http.StatusOK, discoveryDoc)
}
