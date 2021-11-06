package mattermost

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
)

type MattermostHandler struct {
	botUser             *model.User
	client              *model.Client4
	maintainerUsernames []string
	devOpsChannelName   string
}

func NewMattermostHandler(botUsername string, botUser *model.User, client *model.Client4, maintainerUsernames []string, devOpsChannelName string) *MattermostHandler {
	handler :=  &MattermostHandler{
		botUser:             botUser,
		client:              client,
		maintainerUsernames: maintainerUsernames,
		devOpsChannelName:   devOpsChannelName,
	}

	if err := handler.setupUser(botUsername); err != nil {
		panic(err)
	}

	return handler
}

func (m *MattermostHandler) setupUser(username string) error {
	m.botUser.IsBot = true
	m.botUser.Username = username

	if _, resp := m.client.UpdateUser(m.botUser); resp.Error != nil {
		return fmt.Errorf("cannot update user: %w", resp.Error)
	}

	return nil
}

func (m *MattermostHandler) SendError(message string) error {
	for _, username := range m.maintainerUsernames {
		user, resp := m.client.GetUserByUsername(username, "")

		if resp.Error != nil {
			return fmt.Errorf("cannot get user with name %v: %w", username, resp.Error)
		}

		channel, resp := m.client.CreateDirectChannel(m.botUser.Id, user.Id)

		if resp.Error != nil {
			return fmt.Errorf("cannot create direct channel with user %v: %w", user.Username, resp.Error)
		}

		post := &model.Post{}
		post.ChannelId = channel.Id
		post.Message = message

		if _, resp := m.client.CreatePost(post); resp.Error != nil {
			return fmt.Errorf("cannot create post: %w", resp.Error)
		}
	}

	return nil
}

func (m *MattermostHandler) SendWarning(message string) error {
	panic("Implement me!")
}
