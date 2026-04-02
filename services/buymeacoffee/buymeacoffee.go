// Package buymeacoffee implements a Service capable of processing webhooks from buymeacoffee.com
package buymeacoffee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go-avbot/types"

	"github.com/AVENTER-UG/gomatrix"
	"github.com/AVENTER-UG/util/util"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// ServiceType of the Wekan service.
const ServiceType = "buymeacoffee"

// DefaultTemplate contains the template that will be used if none is supplied.
// This matches the default mentioned at: https://docs.travis-ci.com/user/notifications#Customizing-slack-notifications
const DefaultTemplate = (`%{boardsitory}#%{build_number} (%{branch} - %{commit} : %{author}): %{message}
	Change view : %{compare_url}
	Build details : %{build_url}`)

var httpClient = &http.Client{}

// Service contains the Config fields for the Wekan service.
//
// This service will send notifications into a Matrix room when Wekan sends
// webhook events to it. It requires a public domain which Wekan can reach.
// Notices will be sent as the service user ID.
//
// Example JSON request:
//
//	{
//	    rooms: {
//	        "!ewfug483gsfe:localhost": {
//	            boards: {
//	                "1" {
//	                }
//	            }
//	        }
//	    }
//	}
type Service struct {
	types.DefaultService
	webhookEndpointURL string
	RoomID             string
}

type MembershipCancelledEvent struct {
	Type     string         `json:"type"`
	LiveMode bool           `json:"live_mode"`
	Attempt  int            `json:"attempt"`
	Created  int64          `json:"created"`
	EventID  int            `json:"event_id"`
	Data     MembershipData `json:"data"`
}

type MembershipData struct {
	ID                  int64  `json:"id"`
	Amount              int    `json:"amount"`
	Object              string `json:"object"`
	Paused              string `json:"paused"`   // "false"/"true" im Payload
	Status              string `json:"status"`   // "active" / "canceled"
	Canceled            string `json:"canceled"` // "true"/"false"
	Currency            string `json:"currency"`
	PSPID               string `json:"psp_id"`
	DurationType        string `json:"duration_type"`
	MembershipLevelID   int    `json:"membership_level_id"`
	MembershipLevelName string `json:"membership_level_name"`
	StartedAt           int64  `json:"started_at"`

	// NULL‑fähige Felder
	CanceledAt        *int64  `json:"canceled_at"`        // bei "started" -> null, bei "cancelled" -> timestamp
	SupporterFeedback *string `json:"supporter_feedback"` // bei "started" -> fehlt/NULL, bei "cancelled" -> String

	NoteHidden         bool   `json:"note_hidden"`
	SupportNote        string `json:"support_note"`
	SupporterName      string `json:"supporter_name"`
	SupporterID        int    `json:"supporter_id"`
	SupporterEmail     string `json:"supporter_email"`
	CurrentPeriodEnd   int64  `json:"current_period_end"`
	CurrentPeriodStart int64  `json:"current_period_start"`
}

// OnReceiveWebhook receives requests from buymeacoffee.com and sends notifications to Matrix.
func (e *Service) OnReceiveWebhook(w http.ResponseWriter, req *http.Request, client *gomatrix.Client) {
	logrus.Info("Receive buymeacoffee WebHook")

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		logrus.Error("buymeacoffee webhook is missing payload= form value", err)
		w.WriteHeader(400)
		return
	}

	logrus.Debug(string(payload))

	var evt MembershipCancelledEvent
	if err := json.NewDecoder(req.Body).Decode(&evt); err != nil {
		logrus.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch evt.Type {
	case "membership.started":
		logrus.Info("New subscription:", evt.Data.SupporterEmail)

	case "membership.cancelled":
		logrus.Info("Subscription cancelled:", evt.Data.SupporterEmail)
	}

	message := fmt.Sprintf(
		"<strong>Username:</strong> %s<br>"+
			"<strong>Email:</strong> %s<br>"+
			"<strong>Note:</strong> %s",
		evt.Data.SupporterName,
		evt.Data.SupporterEmail,
		evt.Data.SupportNote,
	)

	msg := gomatrix.HTMLMessage{
		Body:          message,
		MsgType:       "m.notice",
		Format:        "org.matrix.custom.html",
		FormattedBody: util.MarkdownRender(message),
	}

	if _, err := client.SendMessageEvent(e.RoomID, "m.room.message", msg); err != nil {
		logrus.WithField("room_id", e.RoomID).Error("Failed to send unifi ring notification to room.")
	}

	w.WriteHeader(200)
}

// Register makes sure the Config information supplied is valid.
func (e *Service) Register(oldService types.Service, client *gomatrix.Client) error {
	logrus.Infof("Buymeacoffee WebhookURL: %s", e.webhookEndpointURL)
	e.joinRooms(client)
	return nil
}

func (e *Service) joinRooms(client *gomatrix.Client) {
	if _, err := client.JoinRoom(e.RoomID, "", nil); err != nil {
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"room_id":    e.RoomID,
			"user_id":    client.UserID,
		}).Error("Failed to join room")
	}
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &Service{
			DefaultService:     types.NewDefaultService(serviceID, serviceUserID, ServiceType),
			webhookEndpointURL: webhookEndpointURL,
		}
	})
}
