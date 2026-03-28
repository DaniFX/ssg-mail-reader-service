package middleware

import (
	"net/http"

	"github.com/DaniFX/ssg-mail-reader-service/internal/models"
	"github.com/gin-gonic/gin"
)

// ImapCredentials contiene i dati estratti dagli header
type ImapCredentials struct {
	Host     string
	Username string
	Password string
}

// RequireImapHeaders è il middleware che verifica la presenza delle credenziali IMAP
func RequireImapHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.GetHeader("X-Imap-Host")
		user := c.GetHeader("X-Imap-User")
		pass := c.GetHeader("X-Imap-Pass")

		// Verifica che tutti e tre gli header siano presenti
		if host == "" || user == "" || pass == "" {
			c.JSON(
				http.StatusBadRequest,
				models.NewErrorResponse(
					"MISSING_IMAP_CREDENTIALS",
					"È necessario fornire gli header X-Imap-Host, X-Imap-User e X-Imap-Pass",
					nil,
				),
			)
			c.Abort() // Blocca l'esecuzione della catena (non chiama l'handler successivo)
			return
		}

		// Salviamo le credenziali nel contesto di Gin per renderle disponibili agli handler
		creds := ImapCredentials{
			Host:     host,
			Username: user,
			Password: pass,
		}
		c.Set("imapCreds", creds)

		c.Next() // Passa il controllo all'handler effettivo
	}
}
