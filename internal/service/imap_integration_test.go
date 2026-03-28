//go:build integration

package service

import (
	"os"
	"testing"

	"github.com/DaniFX/ssg-mail-reader-service/internal/models"
)

func TestArubaPecIntegration(t *testing.T) {
	// Carichiamo le credenziali reali della tua PEC
	host := os.Getenv("MAIL_IMAP_HOST")
	user := os.Getenv("MAIL_IMAP_USER")
	pass := os.Getenv("MAIL_IMAP_PASS")

	if host == "" || user == "" || pass == "" {
		t.Fatal("ERRORE: Variabili d'ambiente (MAIL_IMAP_*) non impostate.")
	}

	svc := NewMailService(host, user, pass)

	// Test 1: Connessione e Lista Cartelle
	folders, err := svc.ListFolders()
	if err != nil {
		t.Fatalf("Fallita connessione IMAP alla PEC: %v", err)
	}

	t.Logf("Connessione riuscita! Cartelle PEC trovate: %v", folders)

	// Test 2: Ricerca Fatture (opzionale, non fallisce se non ne trova)
	t.Log("Eseguo test di ricerca email generica...")
	previews, err := svc.Search(models.SearchCriteria{Folder: "INBOX"})
	if err != nil {
		t.Errorf("Errore durante la ricerca: %v", err)
	}
	t.Logf("Trovate %d email nella INBOX", len(previews))
}
