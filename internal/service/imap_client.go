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

// connect stabilisce la connessione TLS sicura specifica per server PEC/Aruba
func (s *MailService) connect() (*client.Client, error) {
	// Estraiamo l'host senza porta per la validazione del certificato TLS
	hostOnly := s.Host
	if i := strings.Index(hostOnly, ":"); i != -1 {
		hostOnly = hostOnly[:i]
	}

	// Configurazione TLS ottimizzata per i requisiti di sicurezza dei provider PEC
	tlsConfig := &tls.Config{
		ServerName: hostOnly,
		MinVersion: tls.VersionTLS12,
	}

	c, err := client.DialTLS(s.Host, tlsConfig)
	if err != nil {
		return nil, err
	}

	if err := c.Login(s.Username, s.Password); err != nil {
		c.Logout()
		return nil, err
	}

	return c, nil
}

// ListFolders recupera tutte le cartelle della casella PEC
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

// Search restituisce le email filtrate (utile per cercare fatture elettroniche)
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

	_, err = c.Select(folder, true) // Read-only
	if err != nil {
		return nil, err
	}

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

	uids, err := c.UidSearch(searchCriteria)
	if err != nil {
		return nil, err
	}

	if len(uids) == 0 {
		return []models.EmailPreview{}, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	var previews []models.EmailPreview
	for msg := range messages {
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

	return previews, <-done
}

// GetMessage estrae il contenuto e identifica allegati XML (Fatture Elettroniche)
func (s *MailService) GetMessage(folder string, uid uint32) (*models.EmailDetail, error) {
	c, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if _, err := c.Select(folder, true); err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}, messages)
	}()

	msg := <-messages
	if msg == nil {
		return nil, errors.New("messaggio non trovato")
	}

	from := ""
	if len(msg.Envelope.From) > 0 {
		from = msg.Envelope.From[0].Address()
	}

	detail := &models.EmailDetail{
		EmailPreview: models.EmailPreview{
			UID:     msg.Uid,
			Subject: msg.Envelope.Subject,
			From:    from,
			Date:    msg.Envelope.Date,
		},
	}

	r := msg.GetBody(section)
	if r == nil {
		return detail, <-done
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return detail, <-done
	}

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := io.ReadAll(p.Body)
			contentType, _, _ := h.ContentType()
			if contentType == "text/html" {
				detail.HTMLBody = string(b)
			} else {
				detail.Body = string(b)
			}
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			fnLower := strings.ToLower(filename)

			// Identificazione Fattura Elettronica (.xml o firmata .p7m)
			if strings.HasSuffix(fnLower, ".xml") || strings.HasSuffix(fnLower, ".p7m") {
				// Logica predisposta per il salvataggio su Cloud Storage o analisi XML
				// fmt.Printf("Trovata fattura: %s\n", filename)
			}
		}
	}

	return detail, <-done
}

func (s *MailService) CreateFolder(name string) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()
	return c.Create(name)
}

func (s *MailService) MoveMessage(sourceFolder string, uid uint32, destFolder string) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()
	_, err = c.Select(sourceFolder, false)
	if err != nil {
		return err
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	return c.UidMove(seqset, destFolder)
}

func (s *MailService) DeleteMessage(folder string, uid uint32) error {
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Logout()
	_, err = c.Select(folder, false)
	if err != nil {
		return err
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	return c.UidStore(seqset, item, flags, nil)
}
