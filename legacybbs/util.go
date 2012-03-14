package legacybbs

import (
	"appengine"
	"appengine/mail"
	"os"
	"io/ioutil"
	"json"
	"http"
	"bytes"
	"template"
)

type MailConfig struct {
	Sender string
	To string
}

// Retrieves configuration for sending post notification mail
func ParseConfig(file string) (MailConfig, os.Error) {
	var config MailConfig
	jsonString, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(jsonString, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

// Defines task in TaskQueue
func task(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	
	var entry Entry
	entry.Name = r.FormValue("name")
	entry.Title = r.FormValue("title")
	entry.Message = r.FormValue("message")
	
	if err := SendMail(c, entry); err != nil {
		c.Errorf("Mail sending error: " + err.String())
	}
}

// Sends post notification mail
func SendMail(c appengine.Context, entry Entry) os.Error {
	config, err := ParseConfig("./config/mailConfig.json")
	if err != nil {
		return err
	}
	
	// Prepares email message
	msg := new(mail.Message)
	msg.Sender = config.Sender
	msg.To = make([]string, 1)
	msg.To[0] = config.To
	msg.Subject = "New post made from Legacy-BBS-Go"
	var body bytes.Buffer
	var mailTemplate = template.Must(template.New("mail").ParseFile("template/notificationMailTemplate.txt"))
	if err := mailTemplate.Execute(&body, entry); err != nil {
		return err
	}
	msg.Body = body.String()
	if err := mail.Send(c, msg); err != nil {
		return err
	}
	
	c.Infof("Notification mail sent to \"" + config.To + "\"")
	
	return nil
}