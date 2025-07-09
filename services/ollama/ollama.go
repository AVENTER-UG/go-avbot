// Package ollama implements a Service which ollamaes back !commands.
package ollama

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"io"
	"net/http"
	"strings"

	"go-avbot/types"

	"github.com/AVENTER-UG/gomatrix"
	"github.com/AVENTER-UG/util/util"
	"github.com/ollama/ollama/api"
	"github.com/sirupsen/logrus"
)

// ServiceType of the Echo service
const ServiceType = "ollama"

// Service represents the Echo service. It has no Config fields.
type Service struct {
	types.DefaultService
	Host        string
	Port        int
	Model       string
	ContextSize int
}

var ollama *api.Client
var ctx context.Context
var ollamaContext map[string][]int
var image bool

func (e *Service) Register(oldService types.Service, client *gomatrix.Client) error {
	ollamaContext = make(map[string][]int)

	os.Setenv("OLLAMA_HOST", e.Host)
	os.Setenv("OLLAMA_PORT", strconv.Itoa(e.Port))

	var err error

	ollama, err = api.ClientFromEnvironment()
	ctx = context.Background()
	if err != nil {
		return fmt.Errorf("Failed to create a ollama client: %s", err.Error())
	}
	return nil
}

// RawMessage supported:
//
// Responds to every message
func (e *Service) RawMessage(cli *gomatrix.Client, event *gomatrix.Event, body string) {
	if event.Content["msgtype"] != "m.text" {
		return
	}

	bodyLower := strings.ToLower(body)

	rMembers, err := cli.JoinedMembers(event.RoomID)
	if err != nil {
		logrus.WithField("room_id", event.RoomID).Errorf("Could not get room members: %s", err.Error())
		return
	}

	if len(rMembers.Joined) > 2 {
		member := rMembers.Joined[cli.UserID]
		if member.DisplayName != nil {
  		if strings.Contains(bodyLower, *member.DisplayName) {
				e.chat(cli, event.RoomID, e.Model, body, event)
			}
		}
	}

	if len(rMembers.Joined) <= 2 {
		e.chat(cli, event.RoomID, e.Model, body, event)
	}
}

func (e *Service) GetPreviousMessage(cli *gomatrix.Client, roomID, eventID string) (*gomatrix.Event, error) {
	path := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/context/%s", cli.HomeserverURL.String(), roomID, eventID)

	var resp struct {
		Event        gomatrix.Event   `json:"event"`
		EventsBefore []gomatrix.Event `json:"events_before"`
		EventsAfter  []gomatrix.Event `json:"events_after"`
	}

	err := cli.MakeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("Could not create context message: %w", err)
	}

	// Iteriere rückwärts durch die vorherigen Events
	for i := 0; i < len(resp.EventsBefore); i++ {
		event := resp.EventsBefore[i]

		if event.Sender != cli.UserID && event.Type == "m.room.message" {
			return &event, nil
		}
	}

	return nil, fmt.Errorf("Could not find previous image message")
}


func (e *Service) getImage(cli *gomatrix.Client, roomID string, content map[string]interface{}) []byte {
	url, _ := cli.MXCToHTTP(content["url"].(string))

	resp, err := http.Get(url)
	if err != nil {
		logrus.WithField("room_id", roomID).Errorf("Failes to get image data: %s", err.Error())
		return []byte{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.WithField("room_id", roomID).Errorf("Get Image status is not ok: %s", err.Error())
		return []byte{}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithField("room_id", roomID).Errorf("Could not read image data: %s", err.Error())
		return []byte{}
	}

	return data
}

func (e *Service) chat(cli *gomatrix.Client, roomID, model, message string, event *gomatrix.Event) {
	cli.UserTyping(roomID, true, 900000)

  lastEvent, err := e.GetPreviousMessage(cli, roomID, event.ID)
	if err != nil {
		fmt.Errorf("Failes to get last event: %s", err.Error())
	}

	if lastEvent.Content["msgtype"].(string) == "m.text" {
		image = false
	}

	req := &api.GenerateRequest{
		Model:   model,
		Prompt:  message,
		Context: ollamaContext[roomID],
		Stream:  util.BoolToPointer(false),
	}

	if lastEvent.Content["msgtype"].(string) == "m.image" {
		image = true

		data := e.getImage(cli, roomID, lastEvent.Content)
		if data != nil {
			req = &api.GenerateRequest{
				Model:   model,
				Prompt:  message,
				Images:  []api.ImageData{data},
				Context: ollamaContext[roomID],
				Stream:  util.BoolToPointer(false),
			}
		}
	}

	respFunc := func(resp api.GenerateResponse) error {
		if !image {
			if len(ollamaContext[roomID]) >= e.ContextSize {
				// keep only the last 100 items
				ollamaContext[roomID] = ollamaContext[roomID][len(ollamaContext[roomID])-100:]
			}
			ollamaContext[roomID] = append(ollamaContext[roomID], resp.Context...)
		}

		msg := gomatrix.HTMLMessage{
			Body:          resp.Response,
			MsgType:       "m.notice",
			Format:        "org.matrix.custom.html",
			FormattedBody: util.MarkdownRender(resp.Response),
		}

		cli.UserTyping(roomID, false, 3000)

		if _, err := cli.SendMessageEvent(roomID, "m.room.message", msg); err != nil {
			return fmt.Errorf("Failes send event message to matrix: %s", err.Error())
		}
		return nil
	}

	err = ollama.Generate(ctx, req, respFunc)
	if err != nil {
		logrus.WithField("room_id", roomID).Errorf("%s", err.Error())
	}
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &Service{
			DefaultService: types.NewDefaultService(serviceID, serviceUserID, ServiceType),
		}
	})
}
