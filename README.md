# Mail Reader Service

Questo microservizio Go gestisce la lettura, la ricerca e lo spostamento delle email interfacciandosi con qualsiasi server IMAP. Fa parte dell'architettura a microservizi (API Gateway + Cloud Run).

È progettato per essere **stateless**: non mantiene connessioni aperte in background, ma istanzia il client IMAP al volo per ogni richiesta, rendendolo perfetto per lo scaling orizzontale su Google Cloud Run.

## Prerequisiti

* Go 1.21+
* Docker (per la containerizzazione)


## Sviluppo Locale

1. Installa le dipendenze:
   ```bash
   go mod tidy