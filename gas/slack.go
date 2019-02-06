package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/hatajoe/hooks"
	"github.com/hatajoe/hooks/slack"
	slackAPI "github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func setupSlackEventAPIHandlers(cli *slackAPI.Client, db *sql.DB, secret, pattern string) {
	dispatcher := hooks.NewDispatcher(&slack.EventParser{
		ChallengeEventType: "challenge",
		VerifyToken:        true,
		VerificationToken:  secret,
	})

	dispatcher.On("challenge", slack.ChallengeHandler)
	dispatcher.On("app_mention", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			log.Printf(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if appMentionEvent, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.AppMentionEvent); !ok {
			log.Printf("Unexpected event type detected: *slackevents.AppMentionEvent is expected")
			http.Error(w, "Unexpected event type detected: *slackevents.AppMentionEvent is expected", http.StatusInternalServerError)
			return
		} else {
			if err := postSlack(db, cli, appMentionEvent); err != nil {
				log.Printf(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	})

	http.Handle(pattern, dispatcher)
}

func postSlack(db *sql.DB, cli *slackAPI.Client, appMentionEvent *slackevents.AppMentionEvent) error {
	text := appMentionEvent.Text
	args := strings.Split(text, " ")

	if len(args) != 3 || args[1] != "stats" {
		return fmt.Errorf("The mention must contain `stats` and repository name: %v", args)
	}

	repo := args[2]

	pullRequestCount, err := getCount(db, repo, "pull_request")
	if err != nil {
		return err
	}

	pullRequestReviewCount, err := getCount(db, repo, "pull_request_review")
	if err != nil {
		return err
	}

	pullRequestReviewCommentCount, err := getCount(db, repo, "pull_request_review_comment")
	if err != nil {
		return err
	}

	msgOptText := slackAPI.MsgOptionText(fmt.Sprintf(`
total pull-request count is %d
total pull-request-review count is %d
total pull-request-review-comment count is %d
	`, pullRequestCount, pullRequestReviewCount, pullRequestReviewCommentCount), true)
	if _, _, err := cli.PostMessage(appMentionEvent.Channel, msgOptText); err != nil {
		return fmt.Errorf("can't post message: channel=%s, err=%v", appMentionEvent.Channel, err)
	}

	return nil
}

func getCount(db *sql.DB, repo, table string) (int, error) {
	rows, err := db.Query("SELECT COUNT(*) as cnt FROM "+table+" WHERE repository = ?", repo)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cnt := 0
	for rows.Next() {
		if err := rows.Scan(&cnt); err != nil {
			return 0, err
		}
	}
	return cnt, nil
}
