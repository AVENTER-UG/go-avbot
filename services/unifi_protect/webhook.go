// Package unifi_protect_alarm implements a Service capable of processing webhooks from Wekan
package unifi_protect

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/AVENTER-UG/gomatrix"
	"github.com/AVENTER-UG/util/util"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type webhookNotification struct {
	Alarm struct {
		Name       string        `json:"name"`
		Sources    []interface{} `json:"sources"`
		Conditions []struct {
			Condition struct {
				Type   string `json:"type"`
				Source string `json:"source"`
			} `json:"condition"`
		} `json:"conditions"`
		Triggers []struct {
			Key       string `json:"key"`
			Device    string `json:"device"`
			EventID   string `json:"eventId"`
			Timestamp int64  `json:"timestamp"`
		} `json:"triggers"`
		Thumbnail      string `json:"thumbnail"`
		EventPath      string `json:"eventPath"`
		EventLocalLink string `json:"eventLocalLink"`
	} `json:"alarm"`
	Timestamp int64 `json:"timestamp"`
}

// OnReceiveWebhook receives requests from unifi protect and possibly sends requests to Matrix as a result.
// Go-AVBOT cannot register with unifi_protect_alarm for webhooks automatically. The user must manually add the
// webhook endpoint URL.
//
//	notifications:
//	    webhooks: http://go-avbot-endpoint.com/unifi_protect_alarm_webhook_service
func (e *Service) OnReceiveWebhook(w http.ResponseWriter, req *http.Request, client *gomatrix.Client) {
	logrus.Info("Receive Unifi Protect WebHook")

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Error("Unifi webhook is missing payload= form value", err)
		w.WriteHeader(400)
		return
	}

	//logrus.Info(string(payload))

	var notif webhookNotification
	if err := json.Unmarshal([]byte(payload), &notif); err != nil {
		logrus.WithError(err).Error("Unifi webhook received an invalid JSON payload=", payload)
		w.WriteHeader(400)
		return
	}

	message := "<i>Alarm</i> triggerd by: "
	for _, key := range notif.Alarm.Triggers {
		message += key.Key + " "
	}

	msg := gomatrix.HTMLMessage{
		Body:          message,
		MsgType:       "m.notice",
		Format:        "org.matrix.custom.html",
		FormattedBody: util.MarkdownRender(message),
	}

	if _, err := client.SendMessageEvent(e.RoomID, "m.room.message", msg); err != nil {
		logrus.WithField("room_id", e.RoomID).Error("Failed to send unifi ring notification to room.")
	}

	out, length, err := e.GetImageFromThumbnail(notif.Alarm.Thumbnail)
	if err != nil {
		logrus.WithField("room_id", e.RoomID).Error("Could not read thumbnail.", err.Error())
		return
	}

	rmu, err := client.UploadToContentRepo(out, "image/jpeg", length)
	if err != nil {
		logrus.WithField("room_id", e.RoomID).Error("Could not upload thumbnail.", err.Error())
		return
	}
	log.Info(rmu.ContentURI)

	if _, err := client.SendImage(e.RoomID, "file"+notif.Alarm.Triggers[0].EventID+".jpg", rmu.ContentURI); err != nil {
		logrus.WithField("room_id", e.RoomID).Error("Failed to send unifi_protect thumbnail to room.")
	}

	w.WriteHeader(200)
}

func (e *Service) GetImageFromThumbnail(src string) (io.Reader, int64, error) {
	commaIdx := strings.Index(src, ",")
	if commaIdx < 0 {
		return nil, 0, errors.New("invalid data URI: missing comma separator")
	}
	base64Data := src[commaIdx+1:]

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, 0, err
	}

	return bytes.NewReader(decoded), int64(len(decoded)), nil
}
