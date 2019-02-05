package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	models "github.com/google/go-github/v22/github"
	"github.com/hatajoe/hooks"
	"github.com/hatajoe/hooks/github"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	if dbConn, err := sql.Open("sqlite3", os.Getenv("SQLITE_DATABASE")); err != nil {
		log.Fatalf("Error opening sqlite3: %v", err)
	} else {
		db = dbConn
		for _, sql := range []string{
			`CREATE TABLE IF NOT EXISTS "pull_request" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "repository" VARCHAR(255), "sender" VARCHAR(255), "payload" TEXT, "created_at" DATETIME)`,
			`CREATE INDEX IF NOT EXISTS "pull_request_repository" ON "pull_request" ("repository")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_sender" ON "pull_request" ("sender")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_repository_sender" ON "pull_request" ("repository", "sender")`,
			`CREATE TABLE IF NOT EXISTS "pull_request_review" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "repository" VARCHAR(255), "sender" VARCHAR(255), "payload" TEXT, "created_at" DATETIME)`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_repository" ON "pull_request_review" ("repository")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_sender" ON "pull_request_review" ("sender")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_repository_sender" ON "pull_request_review" ("repository", "sender")`,
			`CREATE TABLE IF NOT EXISTS "pull_request_review_comment" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "repository" VARCHAR(255), "sender" VARCHAR(255), "payload" TEXT, "created_at" DATETIME)`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_comment_repository" ON "pull_request_review_comment" ("repository")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_comment_sender" ON "pull_request_review_comment" ("sender")`,
			`CREATE INDEX IF NOT EXISTS "pull_request_review_comment_repository_sender" ON "pull_request_review_comment" ("repository", "sender")`,
		} {
			_, err = db.Exec(sql)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	dispatcher := hooks.NewDispatcher(&github.EventParser{})
	verifier := github.NewVerifyMiddleware(os.Getenv("GITHUB_WEBHOOK_SECRET"))

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
			if err := savePayload(event, payload); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})))
	}

	if err := dispatcher.Listen(os.Getenv("GITHUB_WEBHOOK_SERVER_ENDPOINT"), os.Getenv("GITHUB_WEBHOOK_SERVER_PORT")); err != nil {
		log.Fatal(err)
	}
}

func getPayload(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form failed")
	}
	return r.Form.Get("payload"), nil
}

func savePayload(tableName, payload string) error {
	m := &models.WebHookPayload{}
	if err := json.Unmarshal([]byte(payload), m); err != nil {
		return err
	}
	sql := `INSERT INTO ` + tableName + ` (repository, sender, payload, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(sql, m.GetRepo().GetFullName(), m.GetSender().GetLogin(), payload, time.Now().Format(time.RFC3339))
	return err
}
