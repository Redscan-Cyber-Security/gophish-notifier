package main

import (
	"fmt"
	"encoding/json"
	"html/template"
	"net/url"
	"strings"
	"github.com/ashwanthkumar/slack-go-webhook"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Can't be const because need reference to variable for Slack webhook title
var (
	ClickedLink   string = "Clicked Link"
	SubmittedData string = "Submitted Data"
	EmailOpened   string = "Email Opened"
)

type Sender interface {
	SendSlack() error
	SendTeams() error
	SendEmail() error
}

func senderDispatch(status string, webhookResponse WebhookResponse, response []byte) (Sender, error) {
	if status == ClickedLink {
		return NewClickDetails(webhookResponse, response)
	}
	if status == EmailOpened {
		return NewOpenedDetails(webhookResponse, response)
	}
	if status == SubmittedData {
		return NewSubmittedDetails(webhookResponse, response)
	}
	log.Warn("unknown status:", status)
	return nil, nil
}

// More information about events can be found here:
// https://github.com/gophish/gophish/blob/db63ee978dcd678caee0db71e5e1b91f9f293880/models/result.go#L50
type WebhookResponse struct {
	Success    bool   `json:"success"`
	CampaignID uint   `json:"campaign_id"`
	Message    string `json:"message"`
	Details    string `json:"details"`
	Email      string `json:"email"`
}

func NewWebhookResponse(body []byte) (WebhookResponse, error) {
	var response WebhookResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return WebhookResponse{}, err
	}
	return response, nil
}

type EventDetails struct {
	Payload url.Values        `json:"payload"`
	Browser map[string]string `json:"browser"`
}

func NewEventDetails(detailsRaw []byte) (EventDetails, error) {
	var details EventDetails
	if err := json.Unmarshal(detailsRaw, &details); err != nil {
		return EventDetails{}, err
	}
	return details, nil
}
func (e EventDetails) ID() string {
	return e.Payload.Get("p")
}

func (e EventDetails) Token() string {
        return e.Payload.Get("token")
}

func (e EventDetails) UserAgent() string {
	return e.Browser["user-agent"]
}
func (e EventDetails) Address() string {
	return e.Browser["address"]
}

type SubmittedDetails struct {
	CampaignID uint
	ID         string
	Email      string
	Address    string
	UserAgent  string
	Username   string
	Password   string
	Token	   string
}

func NewSubmittedDetails(response WebhookResponse, detailsRaw []byte) (SubmittedDetails, error) {
	details, err := NewEventDetails(detailsRaw)
	if err != nil {
		return SubmittedDetails{}, err
	}
	submittedDetails := SubmittedDetails{
		CampaignID: response.CampaignID,
		ID:         details.ID(),
		Address:    details.Address(),
		UserAgent:  details.UserAgent(),
		Email:      response.Email,
		Username:   details.Payload.Get("username"),
		Password:   details.Payload.Get("password"),
		Token:	    details.Payload.Get("token"),
	}
	return submittedDetails, nil
}

func (w SubmittedDetails) SendSlack() error {
	red := "#f05b4f"
	attachment := slack.Attachment{Title: &SubmittedData, Color: &red}
	attachment.AddField(slack.Field{Title: "ID", Value: w.ID})
	attachment.AddField(slack.Field{Title: "Address", Value: slackFormatIP(w.Address)})
	attachment.AddField(slack.Field{Title: "User Agent", Value: w.UserAgent})
	if !viper.GetBool("disable_credentials") {
		attachment.AddField(slack.Field{Title: "Email", Value: w.Email})
		attachment.AddField(slack.Field{Title: "Username", Value: w.Username})
		attachment.AddField(slack.Field{Title: "Password", Value: w.Password})
	}
	attachment = addCampaignButton(attachment, w.CampaignID)
	return sendSlackAttachment(attachment)
}

func (w SubmittedDetails) SendTeams() error {
	gophishBase := viper.GetString("base_url")
	url := fmt.Sprintf("%s/campaigns/%d", gophishBase, w.CampaignID)
	color := "#f05b4f"
        msg := "<b>ID:</b> " + w.ID + "<br>" + "<b>IP:</b> " + w.Address + "<br>" + "<b>Email:</b> " + w.Email + "<br>" + "<b>Username:</b> " + w.Username + "<br>" + "<b>Password:</b> " + w.Password + "<br>" + "<b>MFA Token:</b> " + w.Token + "<br>" + "<b>User Agent:</b> " + w.UserAgent
	message := msg + "<br><b>Campaign URL:</b> " + "<a href=%22" + url + "%22>" + url + "</a>"
	return sendTeams("PhishBot - Credentials Submitted", message, color)
}

func (w SubmittedDetails) SendEmail() error {
	templateString := viper.GetString("email_submitted_credentials_template")
        body, err := getEmailBody(templateString, w)
	if err != nil {
		return err
	}
	return sendEmail("PhishBot - Credentials Submitted", body)
}

type ClickDetails struct {
	CampaignID uint
	ID         string
	Email      string
	Address    string
	UserAgent  string
}

func NewClickDetails(response WebhookResponse, detailsRaw []byte) (ClickDetails, error) {
	details, err := NewEventDetails(detailsRaw)
	if err != nil {
		return ClickDetails{}, err
	}
	clickDetails := ClickDetails{
		CampaignID: response.CampaignID,
		ID:         details.ID(),
		Address:    details.Address(),
		Email:      response.Email,
		UserAgent:  details.UserAgent(),
	}
	return clickDetails, nil
}

func (w ClickDetails) SendSlack() error {
	orange := "#ffa500"
	attachment := slack.Attachment{Title: &ClickedLink, Color: &orange}
	attachment.AddField(slack.Field{Title: "ID", Value: w.ID})
	attachment.AddField(slack.Field{Title: "Address", Value: slackFormatIP(w.Address)})
	attachment.AddField(slack.Field{Title: "User Agent", Value: w.UserAgent})
	if !viper.GetBool("slack.disable_credentials") {
		attachment.AddField(slack.Field{Title: "Email", Value: w.Email})
	}
	attachment = addCampaignButton(attachment, w.CampaignID)
	return sendSlackAttachment(attachment)
}

func (w ClickDetails) SendTeams() error {
	color := "#ffa500"
	message := "<b>ID:</b> " + w.ID + "<br>" + "<b>IP:</b> " + w.Address + "<br>" + "<b>Email:</b> " + w.Email + "<br>" + "<b>User Agent:</b> " + w.UserAgent
        return sendTeams("PhishBot - Email Clicked", message, color)
}

func (w ClickDetails) SendEmail() error {
	templateString := viper.GetString("email_send_click_template")
	body, err := getEmailBody(templateString, w)
	if err != nil {
		return err
	}
	return sendEmail("PhishBot - Email Clicked", body)
}

func getEmailBody(templateValue string, obj interface{}) (string, error) {
	out := new(strings.Builder)
	tpl, err := template.New("email").Parse(templateValue)
	if err != nil {
		return "", err
	}
	if err := tpl.Execute(out, obj); err != nil {
		return "", err
	}
	return out.String(), nil
}

type OpenedDetails struct {
	CampaignID uint
	ID         string
	Email      string
	Address    string
	UserAgent  string
}

func NewOpenedDetails(response WebhookResponse, detailsRaw []byte) (OpenedDetails, error) {
	details, err := NewEventDetails(detailsRaw)
	if err != nil {
		return OpenedDetails{}, err
	}
	clickDetails := OpenedDetails{
		CampaignID: response.CampaignID,
		ID:         details.ID(),
		Email:      response.Email,
		Address:    details.Address(),
		UserAgent:  details.UserAgent(),
	}
	return clickDetails, nil
}

func (w OpenedDetails) SendSlack() error {
	yellow := "#ffff00"
	attachment := slack.Attachment{Title: &EmailOpened, Color: &yellow}
	attachment.AddField(slack.Field{Title: "ID", Value: w.ID})
	attachment.AddField(slack.Field{Title: "Address", Value: slackFormatIP(w.Address)})
	attachment.AddField(slack.Field{Title: "User Agent", Value: w.UserAgent})
	if !viper.GetBool("slack.disable_credentials") {
		attachment.AddField(slack.Field{Title: "Email", Value: w.Email})
	}
	attachment = addCampaignButton(attachment, w.CampaignID)
	return sendSlackAttachment(attachment)
}

func (w OpenedDetails) SendTeams() error {
	color := "#ffff00"
	message := "<b>ID:</b> " + w.ID + "<br>" + "<b>IP:</b> " + w.Address + "<br>" + "<b>Email:</b> " + w.Email + "<br>" + "<b>User Agent:</b> " + w.UserAgent
	return sendTeams("PhishBot - Email Opened", message, color)
}


func (w OpenedDetails) SendEmail() error {
	templateString := viper.GetString("email_send_click_template")
	body, err := getEmailBody(templateString, w)
	if err != nil {
		return err
	}
	return sendEmail("PhishBot - Email Opened", body)
}
