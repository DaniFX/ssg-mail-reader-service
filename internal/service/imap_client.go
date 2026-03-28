package service

import (
	"crypto/tls"
	"errors"
	"io"
	"strings"

	"github.com/DaniFX/ssg-mail-reader-service/internal/models"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type MailService struct {
	Host     string
	Username string
	Password string
}

func NewMailService(host, username, password string) *MailService {
	return &MailService{Host: host, Username: username, Password: password}
}

// connect gestisce la connessione e il login in modo sicuro
func (s *MailService) connect() (*client.Client, error) {
	// Aggiungi la porta se non presente (default IMAPS 993)
	hostPort := s.Host
	if !strings.Contains(hostPort, ":") {
		hostPort += ":993"
	}

	c, err := client.DialTLS(hostPort, &tls.Config{ServerName: s.Host})
	if err != nil {
		return nil, err
	}

	if err := c.Login(s.Username, s.Password); err != nil {
		c.Logout()
		return nil, err
	}

	return c, nil
}

// ListFolders recupera tutte le cartelle della casella
func (s *MailService) ListFolders() ([]string, error) {
	c, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var folders []string
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}

	if err := <-done; err != nil {
		return nil, err
	}
	return folders, nil
}

// CreateFolder crea una nuova cartella
func (s *MailService) CreateFolder(name string) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()

	return c.Create(name)
}

// Search restituisce una lista di anteprime basate sui filtri
func (s *MailService) Search(criteria models.SearchCriteria) ([]models.EmailPreview, error) {
	c, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	folder := criteria.Folder
	if folder == "" {
		folder = "INBOX"
	}

	_, err = c.Select(folder, true) // true = ReadOnly
	if err != nil {
		return nil, err
	}

	// Costruisci i criteri di ricerca IMAP
	searchCriteria := imap.NewSearchCriteria()
	if criteria.Subject != "" {
		searchCriteria.Header.Add("Subject", criteria.Subject)
	}
	if criteria.From != "" {
		searchCriteria.Header.Add("From", criteria.From)
	}
	if criteria.BodyContains != "" {
		searchCriteria.Body = []string{criteria.BodyContains}
	}
	if len(searchCriteria.Header) == 0 && len(searchCriteria.Body) == 0 {
		searchCriteria.WithoutFlags = []string{imap.DeletedFlag} // Default: prendi le non eliminate
	}

	uids, err := c.UidSearch(searchCriteria)
	if err != nil {
		return nil, err
	}

	if len(uids) == 0 {
		return []models.EmailPreview{}, nil
	}

	// Recupera l'Envelope (Subject, From, Date) per gli UID trovati
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	var previews []models.EmailPreview
	for msg := range messages {
		if msg.Envelope != nil {
			from := ""
			if len(msg.Envelope.From) > 0 {
				from = msg.Envelope.From[0].Address()
			}
			previews = append(previews, models.EmailPreview{
				UID:     msg.Uid,
				Subject: msg.Envelope.Subject,
				From:    from,
				Date:    msg.Envelope.Date,
			})
		}
	}

	if err := <-done; err != nil {
		return nil, err
	}
	return previews, nil
}

// DeleteMessage aggiunge la flag \Deleted all'email
func (s *MailService) DeleteMessage(folder string, uid uint32) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()

	if folder == "" {
		folder = "INBOX"
	}
	_, err = c.Select(folder, false) // false = ReadWrite
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	// Aggiunge la flag \Deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	return c.UidStore(seqset, item, flags, nil)
}

// MoveMessage sposta un'email tra due cartelle
func (s *MailService) MoveMessage(sourceFolder string, uid uint32, destFolder string) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()

	if sourceFolder == "" {
		sourceFolder = "INBOX"
	}
	_, err = c.Select(sourceFolder, false)
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	// Usa il comando Move (richiede estensione MOVE sul server, se fallisce fa Copy+Delete)
	return c.UidMove(seqset, destFolder)
}

// GetMessage recupera l'email completa (incluso il corpo in HTML e Testo) tramite UID
func (s *MailService) GetMessage(folder string, uid uint32) (*models.EmailDetail, error) {
	c, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if folder == "" {
		folder = "INBOX"
	}
	_, err = c.Select(folder, true) // true = ReadOnly, non modifichiamo lo stato dell'email
	if err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	// Definiamo cosa vogliamo scaricare: il corpo intero (BODY[]) e le intestazioni (Envelope)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, items, messages)
	}()

	var imapMsg *imap.Message
	for m := range messages {
		imapMsg = m
	}

	if err := <-done; err != nil {
		return nil, err
	}

	if imapMsg == nil {
		return nil, errors.New("email non trovata")
	}

	// Costruiamo la risposta di base (Mittente, Oggetto, Data)
	from := ""
	if len(imapMsg.Envelope.From) > 0 {
		from = imapMsg.Envelope.From[0].Address()
	}

	detail := &models.EmailDetail{
		EmailPreview: models.EmailPreview{
			UID:     imapMsg.Uid,
			Subject: imapMsg.Envelope.Subject,
			From:    from,
			Date:    imapMsg.Envelope.Date,
		},
	}

	// Recuperiamo il reader per il corpo del messaggio
	bodyReader := imapMsg.GetBody(section)
	if bodyReader == nil {
		// Se il corpo è vuoto, restituiamo comunque i dati dell'intestazione
		return detail, nil
	}

	// Inizializziamo il parser MIME di go-message
	mr, err := mail.CreateReader(bodyReader)
	if err != nil {
		return detail, nil // Se il parsing fallisce, restituiamo almeno l'intestazione
	}

	// Iteriamo su tutte le "parti" del messaggio (Testo, HTML, eventuali allegati in linea)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break // Abbiamo finito di leggere tutte le parti
		} else if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// Si tratta del testo del messaggio
			b, _ := io.ReadAll(p.Body)
			contentType, _, _ := h.ContentType()

			// A seconda del content type, popoliamo il campo HTML o Plain Text
			if contentType == "text/html" {
				detail.HTMLBody = string(b)
			} else if strings.HasPrefix(contentType, "text/plain") {
				detail.Body = string(b)
			}
		case *mail.AttachmentHeader:
			// Questa è la sezione in cui, in futuro, potrai gestire il salvataggio degli allegati
			// filename, _ := h.Filename()
			// log.Printf("Trovato allegato: %s", filename)
			continue
		}
	}

	return detail, nil
}
