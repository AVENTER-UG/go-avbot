// Package ollama implements a Service which ollamaes back !commands.
package ollama

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

// Commands supported:
//
//	!ollama some message
//
// Responds with a notice of "some message".
func (e *Service) Commands(cli *gomatrix.Client) []types.Command {
	return []types.Command{
		{
			Path: []string{"ollama"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				e.chat(cli, roomID, strings.Join(args, " "))
				return nil, nil
			},
		},
		{
			Path: []string{"ollama", "model"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return e.setModel(args)
			},
		},
	}
}

func (e *Service) setModel(args []string) (interface{}, error) {
	if len(args) < 1 {
		return &gomatrix.TextMessage{MsgType: "m.notice", Body: fmt.Sprintf(`Missing parameters.`)}, nil
	}

	e.Model = args[0]

	return &gomatrix.TextMessage{MsgType: "m.notice", Body: "Set Model to: " + e.Model}, nil
}

func (e *Service) chat(cli *gomatrix.Client, roomID, message string) {
	cli.UserTyping(roomID, true, 900000)

	req := &api.GenerateRequest{
		Model:   e.Model,
		Prompt:  message,
		Context: ollamaContext[roomID],
		Stream:  util.BoolToPointer(false),
	}

	respFunc := func(resp api.GenerateResponse) error {
		if len(ollamaContext[roomID]) >= e.ContextSize {
			// keep only the last 100 items
			ollamaContext[roomID] = ollamaContext[roomID][len(ollamaContext[roomID])-100:]
		}
		ollamaContext[roomID] = append(ollamaContext[roomID], resp.Context...)

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

	err := ollama.Generate(ctx, req, respFunc)
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
