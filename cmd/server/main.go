package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DaniFX/ssg-mail-reader-service/internal/handlers"
	"github.com/DaniFX/ssg-mail-reader-service/internal/middleware"
	"github.com/gin-gonic/gin"
)

// registerToGateway invia il contratto API al Gateway per registrarsi al proxy
func registerToGateway() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	serviceURL := os.Getenv("SERVICE_URL")
	internalSecret := os.Getenv("INTERNAL_SECRET")

	if gatewayURL == "" || serviceURL == "" || internalSecret == "" {
		log.Println("⚠️ GATEWAY_URL, SERVICE_URL o INTERNAL_SECRET non definiti. Handshake disabilitato.")
		return
	}

	doc := handlers.GetDiscoveryDoc()
	jsonData, err := json.Marshal(doc)
	if err != nil {
		log.Printf("❌ Errore marshal discovery doc: %v", err)
		return
	}

	// Lancia la registrazione in background
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}

		// Tentiamo la registrazione fino a 5 volte (utile se i container partono insieme)
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", gatewayURL+"/internal/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Internal-Secret", internalSecret)
			req.Header.Set("X-Service-Url", serviceURL) // Passiamo l'URL del nostro microservizio

			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					log.Println("✅ Handshake con il Gateway completato con successo!")
					resp.Body.Close()
					return
				}
				log.Printf("⚠️ Handshake rifiutato con status: %d", resp.StatusCode)
				resp.Body.Close()
			} else {
				log.Printf("⚠️ Errore connessione al Gateway: %v", err)
			}

			log.Printf("⏳ Riprovo tra 5 secondi... (tentativo %d/5)", i+1)
			time.Sleep(5 * time.Second)
		}
		log.Println("❌ Impossibile registrarsi al Gateway dopo 5 tentativi.")
	}()
}

func main() {
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Lanciamo l'handshake in background
	registerToGateway()

	v1 := router.Group("/api/v1/mail")
	v1.Use(middleware.RequireImapHeaders())
	{
		v1.GET("/folders", handlers.GetFolders)
		v1.POST("/folders", handlers.CreateFolder)
		v1.POST("/search", handlers.SearchEmails)
		v1.GET("/messages/:uid", handlers.GetMessage)
		v1.PUT("/messages/:uid/move", handlers.MoveMessage)
		v1.DELETE("/messages/:uid", handlers.DeleteMessage)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 mail-reader-service in ascolto sulla porta %s", port)
	if err := router.Run(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Errore durante l'avvio del server: %v", err)
	}
}
