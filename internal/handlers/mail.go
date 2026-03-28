package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/DaniFX/ssg-mail-reader-service/internal/middleware"
	"github.com/DaniFX/ssg-mail-reader-service/internal/models"
	"github.com/DaniFX/ssg-mail-reader-service/internal/service"
)

// Helper per istanziare il servizio IMAP dal contesto della richiesta
func getMailService(c *gin.Context) *service.MailService {
	credsValue, exists := c.Get("imapCreds")
	if !exists {
		return nil
	}
	creds := credsValue.(middleware.ImapCredentials)
	return service.NewMailService(creds.Host, creds.Username, creds.Password)
}

// Helper per parsare l'UID dai parametri dell'URL
func parseUID(c *gin.Context) (uint32, error) {
	uidStr := c.Param("uid")
	uid, err := strconv.ParseUint(uidStr, 10, 32)
	return uint32(uid), err
}

// GET /api/v1/mail/folders
func GetFolders(c *gin.Context) {
	svc := getMailService(c)
	if svc == nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse("UNAUTHORIZED", "Credenziali mancanti", nil))
		return
	}

	folders, err := svc.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_LIST_ERR", err.Error(), nil))
		return
	}
	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{"folders": folders}, nil))
}

// POST /api/v1/mail/folders
func CreateFolder(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "Nome cartella richiesto", nil))
		return
	}

	svc := getMailService(c)
	if err := svc.CreateFolder(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_CREATE_FOLDER_ERR", err.Error(), nil))
		return
	}
	
	c.JSON(http.StatusCreated, models.NewSuccessResponse(gin.H{"message": "Cartella creata con successo"}, nil))
}

// POST /api/v1/mail/search
func SearchEmails(c *gin.Context) {
	var req models.SearchCriteria
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "Payload JSON non valido", nil))
		return
	}

	svc := getMailService(c)
	emails, err := svc.Search(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_SEARCH_ERR", err.Error(), nil))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{"emails": emails}, nil))
}

// GET /api/v1/mail/messages/:uid
func GetMessage(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "UID non valido", nil))
		return
	}
	
	folder := c.DefaultQuery("folder", "INBOX") // Recupera la cartella dalla query string
	svc := getMailService(c)

	// Nota: qui chiamiamo un metodo GetMessage (da implementare in imap_client.go)
	// per estrarre anche il body/HTML. Per ora ritorna un placeholder.
	msg, err := svc.GetMessage(folder, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_FETCH_ERR", err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{"message": msg}, nil))
}

// PUT /api/v1/mail/messages/:uid/move
func MoveMessage(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "UID non valido", nil))
		return
	}

	var req struct {
		DestinationFolder string `json:"destinationFolder" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "DestinationFolder richiesto", nil))
		return
	}

	sourceFolder := c.DefaultQuery("folder", "INBOX")
	svc := getMailService(c)

	if err := svc.MoveMessage(sourceFolder, uid, req.DestinationFolder); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_MOVE_ERR", err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{"message": "Email spostata con successo"}, nil))
}

// DELETE /api/v1/mail/messages/:uid
func DeleteMessage(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("BAD_REQUEST", "UID non valido", nil))
		return
	}

	folder := c.DefaultQuery("folder", "INBOX")
	svc := getMailService(c)

	if err := svc.DeleteMessage(folder, uid); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("IMAP_DELETE_ERR", err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewSuccessResponse(gin.H{"message": "Email eliminata con successo"}, nil))
}