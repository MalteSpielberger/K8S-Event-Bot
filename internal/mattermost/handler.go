package mattermost

import (
	"errors"
	"fmt"
	uuid2 "github.com/golangee/uuid"
	"github.com/mattermost/mattermost-server/v5/model"
	"k8s.io/apimachinery/pkg/types"
	"k8sbot/internal/i18n"
	"k8sbot/internal/reportstorage"
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

func (m *MattermostHandler) SendReport(objectID types.UID, namespaces, reason, resource, message, lastTimeStampStr string, count int32) error {
	var new *reportstorage.Report

	old, err := m.reportStorage.ReadByObjectID(objectID)

	if errors.Is(err, &reportstorage.NoReportErr{}) {
		new = &reportstorage.Report{
			ID:               uuid2.New(),
			ReportedObject:   objectID,
			Namespace:        namespaces,
			Reason:           reason,
			Resource:         resource,
			Msg:              message,
			Count:            count,
			ReportTimes:      1,
			LastTimestampStr: lastTimeStampStr,
			IsInProgress:     false,
		}

		team, resp := m.client.GetTeamByName(m.teamId, "")

		if resp.Error != nil {
			return fmt.Errorf("cannot get team by given name: %w", resp.Error)
		}

		channel, resp := m.client.GetChannelByName(m.devOpsChannelName, team.Id, "")

		if resp.Error != nil {
			return fmt.Errorf("cannot get channel: %w", err)
		}

		post := &model.Post{}
		post.ChannelId = channel.Id

		attachment := []*model.SlackAttachment{{
			Title: m.res.Warning(),
			Text:  m.res.UnexpectedEvent(),
			Fields: []*model.SlackAttachmentField{
				{
					Title: m.res.Namespace(),
					Value: namespaces,
					Short: true,
				},
				{
					Title: m.res.Reason(),
					Value: reason,
					Short: true,
				},
				{
					Title: m.res.Object(),
					Value: resource,
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
			return fmt.Errorf("cannot create post for report: %w", resp.Error)
		} else {
			if err := m.reportStorage.SetPostID(new.ID, post.Id); err != nil {
				return fmt.Errorf("cannot set post id for new report: %w", err)
			}
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("cannot read object: %w", err)
	}

	new = old

	if err := m.reportStorage.IncreaseCounter(new.ID); err != nil {
		fmt.Errorf("cannot increase counter of report: %w", err)
	}

	post, resp := m.client.GetPost(old.PostID, "")

	if resp.Error != nil {
		return fmt.Errorf("cannot get post for report: %w", resp.Error)
	}

	if len(post.Attachments()) != 1 {
		return fmt.Errorf("got invalid post")
	}

	attachments := post.Attachments()

	attachments[0].Fields[len(attachments[0].Fields)-1].Value = new.Count

	post.AddProp("attachments", attachments)

	if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
		return fmt.Errorf("cannot updated post: %w", err)
	}

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
