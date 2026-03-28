# ==========================================
# Fase 1: Builder
# ==========================================
FROM golang:1.22-alpine AS builder

# Installa git e i certificati CA (necessari per scaricare le dipendenze e per le chiamate TLS)
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Imposta la cartella di lavoro all'interno del container
WORKDIR /app

# Copia i file delle dipendenze e scaricale
COPY go.mod go.sum ./
RUN go mod download

# Copia il resto del codice sorgente
COPY . .

# Compila l'eseguibile disabilitando il CGO per avere un binario statico al 100%
# Questo lo rende ultra-veloce da avviare su Cloud Run
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/server ./cmd/server

# ==========================================
# Fase 2: Immagine Finale (Leggerissima)
# ==========================================
FROM alpine:latest

# Importa i certificati SSL dal builder (FONDAMENTALE per IMAP su TLS)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Imposta la timezone
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Crea un utente non-root per ragioni di sicurezza
RUN adduser -D -g '' appuser
USER appuser

# Imposta la directory di lavoro
WORKDIR /app

# Copia il binario compilato dalla fase di build
COPY --from=builder /go/bin/server /app/server

# Cloud Run inietta la variabile d'ambiente PORT (default 8080)
ENV PORT=8080

# Esponi la porta
EXPOSE 8080

# Comando di avvio
ENTRYPOINT ["/app/server"]