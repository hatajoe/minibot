package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nlopes/slack"
)

func init() {

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	if db, err := sql.Open("sqlite3", os.Getenv("SQLITE_DATABASE")); err != nil {
		log.Fatalf("Error opening sqlite3: %v", err)
	} else {
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

	db, err := sql.Open("sqlite3", os.Getenv("SQLITE_DATABASE"))
	if err != nil {
		log.Fatalf("Error opening sqlite3: %v", err)
	}

	cli := slack.New(os.Getenv("SLACK_API_TOKEN"), slack.OptionDebug(true))

	setupGitHubWebhookHandlers(db, os.Getenv("GITHUB_WEBHOOK_SECRET"), os.Getenv("GITHUB_WEBHOOK_SERVER_ENDPOINT"))
	setupSlackEventAPIHandlers(cli, db, os.Getenv("SLACK_EVENT_API_SECRET"), os.Getenv("SLACK_EVENT_SERVER_ENDPOINT"))

	log.Fatal(http.ListenAndServe(os.Getenv("SERVER_ADDR"), nil))
}
