package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	models "github.com/google/go-github/v22/github"
	"github.com/hatajoe/hooks"
	"github.com/hatajoe/hooks/github"
)

func setupGitHubWebhookHandlers(db *sql.DB, secret, pattern string) {
	dispatcher := hooks.NewDispatcher(&github.EventParser{})
	verifier := github.NewVerifyMiddleware(secret)

	dispatcher.On("ping", verifier.Verify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	for _, e := range []string{
		"pull_request",
		"pull_request_review",
		"pull_request_review_comment",
	} {
		event := e
		dispatcher.On(event, verifier.Verify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, err := getPayload(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := savePayload(db, event, payload); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})))
	}

	http.Handle(pattern, dispatcher)
}

func getPayload(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form failed")
	}
	return r.Form.Get("payload"), nil
}

func savePayload(db *sql.DB, tableName, payload string) error {
	m := &models.WebHookPayload{}
	if err := json.Unmarshal([]byte(payload), m); err != nil {
		return err
	}
	sql := `INSERT INTO ` + tableName + ` (repository, sender, payload, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(sql, m.GetRepo().GetFullName(), m.GetSender().GetLogin(), payload, time.Now().Format(time.RFC3339))
	return err
}
