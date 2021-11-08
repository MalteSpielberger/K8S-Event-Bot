package mattermost

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"k8s-event-bot/internal/i18n"
	"k8s-event-bot/internal/reportstorage"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"path/filepath"
)

type MattermostHandler struct {
	botUser             *model.User
	client              *model.Client4
	maintainerUsernames []string
	devOpsChannelName   string
	teamId              string
	res                 i18n.Resources
	reportStorage       reportstorage.ReportStorage
}

func NewMattermostHandler(botUsername string, botUser *model.User, client *model.Client4, maintainerUsernames []string, devOpsChannelName, teamId string, storage reportstorage.ReportStorage) *MattermostHandler {
	handler := &MattermostHandler{
		botUser:             botUser,
		client:              client,
		maintainerUsernames: maintainerUsernames,
		devOpsChannelName:   devOpsChannelName,
		teamId:              teamId,
		res:                 i18n.NewResources("de-DE"),
		reportStorage:       storage,
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

func (m *MattermostHandler) IsObjectReported(id types.UID) (bool, error) {
	return m.reportStorage.WasReportedEarlier(id)
}

func (m *MattermostHandler) AddReportToPost(objectId types.UID) error {
	log.Println("add to report!")

	postId, err := m.reportStorage.GetPostId(objectId)

	if err != nil {
		return fmt.Errorf("cannot get postId from storage: %w", err)
	}

	post, resp := m.client.GetPost(postId, "")

	if resp.Error != nil {
		return fmt.Errorf("cannot get post from client: %w", resp.Error)
	}

	if len(post.Attachments()) != 1 {
		return fmt.Errorf("invalid post!")
	}

	lastField := post.Attachments()[0].Fields[len(post.Attachments()[0].Fields)-1]

	// Check if the lastField is used, to show how often
	// the report was updated.
	if lastField.Title != m.res.CountReportFromBot() {
		attachments := post.Attachments()

		attachments[0].Fields = append(attachments[0].Fields, &model.SlackAttachmentField{
			Title: m.res.CountReportFromBot(),
			Value: 1,
			Short: true,
		})

		post.AddProp("attachments", attachments)

		if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
			return fmt.Errorf("cannot updated post: %w", err)
		}
	} else {
		attachments := post.Attachments()

		attachments[0].Fields[len(attachments[0].Fields)-1].Value = attachments[0].Fields[len(attachments[0].Fields)-1].Value.(float64) +1

		post.AddProp("attachments", attachments)

		if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
			return fmt.Errorf("cannot updated post: %w", err)
		}
	}

	/*
		lastField := post.Attachments()[0].Fields[len(post.Attachments()[0].Fields)-1]

		//Check if the report was updated once
		if lastField.Title == m.res.CountReportFromBot() {
			lastField.Value = lastField.Value.(int) + 1
		} else {
			post.Attachments()[0].Fields = append(post.Attachments()[0].Fields, &model.SlackAttachmentField{
				Title: m.res.CountReportFromBot(),
				Value: 1,
				Short: true,
			})
		}

		if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
			return fmt.Errorf("cannot updated post: %w", resp.Error)
		}
	*/

	return nil
}

func (m *MattermostHandler) SendInternalError(err error) {
	team, resp := m.client.GetTeamByName(m.teamId, "")

	if resp.Error != nil {
		log.Fatalln(resp.Error)
	}

	channel, resp := m.client.GetChannelByName(m.devOpsChannelName, team.Id, "")

	if resp.Error != nil {
		log.Fatalln(resp.Error)
	}

	post := &model.Post{}
	post.ChannelId = channel.Id

	attachment := []*model.SlackAttachment{{
		Title:    m.res.InternalError(),
		Text:     err.Error(),
		ThumbURL: filepath.Join("assets", "images", "error.png"),
	}}

	model.ParseSlackAttachment(post, attachment)

	if _, resp := m.client.CreatePost(post); resp.Error != nil {
		log.Fatalln(resp.Error)
	}
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

		actions := []*model.PostAction{}

		actions = append(actions, &model.PostAction{
			Name:  "Check",
			Type:  model.POST_ACTION_TYPE_BUTTON,
			Style: "default",
			Integration: &model.PostActionIntegration{
				URL: "plugins/com.mattermost.server-hello-world",
			},
		})

		attachment := []*model.SlackAttachment{{
			AuthorName: m.botUser.Username,
			Timestamp:  "Error",
			Actions:    actions,
		}}

		model.ParseSlackAttachment(post, attachment)

		if _, resp := m.client.CreatePost(post); resp.Error != nil {
			return fmt.Errorf("cannot create post: %w", resp.Error)
		}
	}

	return nil
}

func (m *MattermostHandler) SendPodRestartWarning(pod, namespace string, restarts int) error {
	team, resp := m.client.GetTeamByName(m.teamId, "")

	if resp.Error != nil {
		fmt.Errorf("cannot get team: %w", resp.Error)
	}

	channel, resp := m.client.GetChannelByName(m.devOpsChannelName, team.Id, "")

	if resp.Error != nil {
		return fmt.Errorf("cannot get DevOps-channel: %w", resp.Error)
	}

	post := &model.Post{}
	post.ChannelId = channel.Id

	attachment := []*model.SlackAttachment{{
		Title: m.res.Warning(),
		Text:  m.res.WarningRestartPod(),
		Fields: []*model.SlackAttachmentField{
			{
				Title: m.res.Pod(),
				Value: pod,
				Short: true,
			},
			{
				Title: m.res.Namespace(),
				Value: namespace,
				Short: true,
			},
			{
				Title: m.res.Restarts(),
				Value: fmt.Sprintf("%v", restarts),
				Short: true,
			},
		},
		ThumbURL: filepath.Join("assets", "images", "warning.png"),
	}}

	model.ParseSlackAttachment(post, attachment)

	if _, resp := m.client.CreatePost(post); resp.Error != nil {
		return fmt.Errorf("cannot create post: %w", resp.Error)
	}

	return nil
}

func (m *MattermostHandler) SendEventWarning(objectId types.UID, namespace, reason, object, message, lastTimeStampStr string, count int32) error {
	wasReported, err := m.IsObjectReported(objectId)

	if err != nil {
		return fmt.Errorf("cannot check if object was reported earlier: %w", err)
	}

	if wasReported {
		if err := m.AddReportToPost(objectId); err != nil {
			return fmt.Errorf("cannot add num to post: %w", err)
		}

		return nil
	}

	team, resp := m.client.GetTeamByName(m.teamId, "")

	if resp.Error != nil {
		fmt.Errorf("cannot get team: %w", resp.Error)
	}

	channel, resp := m.client.GetChannelByName(m.devOpsChannelName, team.Id, "")

	if resp.Error != nil {
		return fmt.Errorf("cannot get DevOps-channel: %w", resp.Error)
	}

	post := &model.Post{}
	post.ChannelId = channel.Id

	attachment := []*model.SlackAttachment{{
		Title: m.res.Warning(),
		Text:  m.res.UnexpectedEvent(),
		Fields: []*model.SlackAttachmentField{
			{
				Title: m.res.Namespace(),
				Value: namespace,
				Short: true,
			},
			{
				Title: m.res.Reason(),
				Value: reason,
				Short: true,
			},
			{
				Title: m.res.Object(),
				Value: object,
				Short: true,
			},
			{
				Title: m.res.Message(),
				Value: message,
				Short: true,
			},
			{
				Title: m.res.Count(),
				Value: count,
				Short: true,
			},
			{
				Title: m.res.LastSeen(),
				Value: lastTimeStampStr,
				Short: true,
			},
		},
	}}

	model.ParseSlackAttachment(post, attachment)

	if post, resp := m.client.CreatePost(post); resp.Error != nil {
		return fmt.Errorf("cannot create post: %w", resp.Error)
	} else {
		if err := m.reportStorage.Add(objectId, post.Id); err != nil {
			return fmt.Errorf("cannot add item to storage: %w", err)
		}
	}

	return nil
}
