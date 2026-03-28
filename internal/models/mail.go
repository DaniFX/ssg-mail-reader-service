package models

import "time"

// SearchCriteria mappa il payload in ingresso per la ricerca
type SearchCriteria struct {
	Folder       string `json:"folder"` // Default "INBOX"
	Subject      string `json:"subject,omitempty"`
	From         string `json:"from,omitempty"`
	BodyContains string `json:"bodyContains,omitempty"`
}

// EmailPreview è la struttura usata per le liste (non contiene il corpo intero per efficienza)
type EmailPreview struct {
	UID     uint32    `json:"uid"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	Date    time.Time `json:"date"`
}

// EmailDetail contiene l'email completa (incluso il corpo)
type EmailDetail struct {
	EmailPreview
	Body     string `json:"body"`
	HTMLBody string `json:"htmlBody,omitempty"`
}
