package main

import (
	"fmt"
	"net/smtp"
	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/spf13/viper"
)

func sendSlackAttachment(attachment slack.Attachment) error {
	payload := slack.Payload{
		Username:    viper.GetString("slack.bot_username"),
		Channel:     viper.GetString("slack.bot_channel"),
		IconEmoji:   viper.GetString("slack.bot_emoji"),
		Attachments: []slack.Attachment{attachment},
	}
	if errs := slack.Send(viper.GetString("slack.webhook"), "", payload); len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func sendTeams(title, body string, color string) error {
	// init the client
	mstClient := goteamsnotify.NewClient()
	// Setup message card
	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Title = title
	msgCard.Text = body
	msgCard.ThemeColor = color

	 if err := mstClient.Send(viper.GetString("teams.webhook"), msgCard); err != nil {
	         return err
	}	
        return nil
}

func sendEmail(subject, body string) error {
	from := viper.GetString("email.sender")
	pass := viper.GetString("email.sender_password")
	to := viper.GetString("email.recipient")
	hostAddr := viper.GetString("email.host_addr")
	host := viper.GetString("email.host")

	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", from, to, subject, body)

	plainAuth := smtp.PlainAuth("", from, pass, host)
	if err := smtp.SendMail(hostAddr, plainAuth, from, []string{to}, []byte(msg)); err != nil {
		return err
	}
	return nil
}
