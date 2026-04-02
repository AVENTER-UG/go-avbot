// Package buymeacoffee implements a Service capable of processing webhooks from buymeacoffee.com
package buymeacoffee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-avbot/database"
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
	Type     string     `json:"type"`
	LiveMode bool       `json:"live_mode"`
	Attempt  int        `json:"attempt"`
	Created  int64      `json:"created"`
	EventID  int        `json:"event_id"`
	Data     Membership `json:"data"`
}

type Membership struct {
	ID                  int    `json:"id"`
	Amount              int    `json:"amount"`
	Object              string `json:"object"`
	Paused              string `json:"paused"`
	Status              string `json:"status"`
	Canceled            string `json:"canceled"`
	Currency            string `json:"currency"`
	PSPID               string `json:"psp_id"`
	DurationType        string `json:"duration_type"`
	MembershipLevelID   int    `json:"membership_level_id"`
	MembershipLevelName string `json:"membership_level_name"`
	StartedAt           int64  `json:"started_at"`
	CanceledAt          *int64 `json:"canceled_at"`
	NoteHidden          bool   `json:"note_hidden"`
	SupportNote         string `json:"support_note"`
	SupporterName       string `json:"supporter_name"`
	SupporterID         int    `json:"supporter_id"`
	SupporterEmail      string `json:"supporter_email"`
	CurrentPeriodEnd    int64  `json:"current_period_end"`
	CurrentPeriodStart  int64  `json:"current_period_start"`
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

	logrus.Info(string(payload))

	var evt MembershipCancelledEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		logrus.WithError(err).Error("Buymeacoffee received an invalid JSON payload=", payload)
		w.WriteHeader(400)
		return
	}

	switch evt.Type {
	case "membership.started":
		logrus.Info("New subscription:", evt.Data.SupporterEmail)

	case "membership.cancelled":
		logrus.Info("Subscription cancelled:", evt.Data.SupporterEmail)

		if err := database.GetServiceDB().DeleteBMCSupporter(evt.Data.SupporterEmail); err != nil {
			message := fmt.Sprintf("<b>Failed to delete supporter: %s</b>", evt.Data.SupporterEmail)

			logrus.WithError(err).Error("Failed to delete supporter")
			msg := gomatrix.HTMLMessage{
				Body:          message,
				MsgType:       "m.notice",
				Format:        "org.matrix.custom.html",
				FormattedBody: util.MarkdownRender(message),
			}
			if _, err := client.SendMessageEvent(e.RoomID, "m.room.message", msg); err != nil {
				logrus.WithField("room_id", e.RoomID).Error("Failed to send buymeacoffee notification to room.")
			}
		}
	}

	message := fmt.Sprintf(
		"%s<br>"+
			"<strong>Username:</strong> %s<br>"+
			"<strong>Email:</strong> %s<br>"+
			"<strong>Note:</strong> %s",
		evt.Type,
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
		logrus.WithField("room_id", e.RoomID).Error("Failed to send buymeacoffee notification to room.")
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

// Commands supported:
//
//	!bmc supporter email <email> <matrix-id> <name>
//	Stores supporter information in the database
//
//	!bmc supporter delete <email>
//	Deletes a supporter from the database
//
//	!bmc some message
//	Responds with a notice of "some message".
func (e *Service) Commands(cli *gomatrix.Client) []types.Command {
	return []types.Command{
		{
			Path: []string{"supporter", "add"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				if len(args) < 3 {
					return &gomatrix.TextMessage{
						MsgType: "m.notice",
						Body:    "Usage: !bmc supporter add <email> <matrix-id> <name>",
					}, nil
				}
				email := args[0]
				matrixID := args[1]
				name := strings.Join(args[2:], " ")

				if err := database.GetServiceDB().StoreBMCSupporter(email, matrixID, name); err != nil {
					return &gomatrix.TextMessage{
						MsgType: "m.notice",
						Body:    "Failed to store supporter: " + err.Error(),
					}, nil
				}

				return &gomatrix.TextMessage{
					MsgType: "m.notice",
					Body:    "Successfully stored supporter: " + name,
				}, nil
			},
		},
		{
			Path: []string{"supporter", "delete"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				if len(args) < 1 {
					return &gomatrix.TextMessage{
						MsgType: "m.notice",
						Body:    "Usage: !bmc supporter delete <email>",
					}, nil
				}
				email := args[0]

				if err := database.GetServiceDB().DeleteBMCSupporter(email); err != nil {
					return &gomatrix.TextMessage{
						MsgType: "m.notice",
						Body:    "Failed to delete supporter: " + err.Error(),
					}, nil
				}

				return &gomatrix.TextMessage{
					MsgType: "m.notice",
					Body:    "Successfully deleted supporter: " + email,
				}, nil
			},
		},
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
