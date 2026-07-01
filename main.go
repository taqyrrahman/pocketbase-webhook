package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	webhookURL    = os.Getenv("WEBHOOK_URL")
	webhookAPIKey = os.Getenv("WEBHOOK_API_KEY")

	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type WebhookPayload struct {
	Source     string         `json:"source"`
	Version    int            `json:"version"`
	Event      string         `json:"event"`
	Collection string         `json:"collection"`
	RecordID   string         `json:"recordId"`
	Timestamp  time.Time      `json:"timestamp"`
	Record     map[string]any `json:"record"`
}

func sendWebhook(event string, e *core.RecordEvent) {
	if webhookURL == "" {
		return
	}

	payload := WebhookPayload{
		Source:     "pocketbase",
		Version:    1,
		Event:      event,
		Collection: e.Record.Collection().Name,
		RecordID:   e.Record.Id,
		Timestamp:  time.Now().UTC(),
		Record:     e.Record.PublicExport(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal webhook payload: %v", err)
		return
	}

	go func() {
		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			log.Printf("failed to create webhook request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		if webhookAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+webhookAPIKey)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("failed to send webhook: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			log.Printf("webhook returned HTTP %d", resp.StatusCode)
		}
	}()
}

func main() {
	app := pocketbase.New()

	app.OnRecordAfterCreateSuccess().BindFunc(func(e *core.RecordEvent) error {
		sendWebhook("record.created", e)
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess().BindFunc(func(e *core.RecordEvent) error {
		sendWebhook("record.updated", e)
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess().BindFunc(func(e *core.RecordEvent) error {
		sendWebhook("record.deleted", e)
		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}