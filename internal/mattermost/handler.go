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
	"time"
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

//TODO: REMOVE REPORT WHEN POST WAS DELETED BY CHAT USER

func (m *MattermostHandler) Listen(done <-chan bool) error {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case _ = <-ticker.C:
				if err := m.checkReports(); err != nil {
					m.SendInternalError(err)
				}
			}
		}
	}()
	return nil
}

func (m *MattermostHandler) checkReports() error {
	reports, err := m.reportStorage.ReadAll()

	if err != nil {
		return fmt.Errorf("cannot read reports: %w", err)
	}

	for _, r := range reports {
		if !r.ReportStopped {
			//Check if someone deletes the post
			// When this ist the case, the report will
			// be removed from the storage.
			post, resp := m.client.GetPost(r.PostID, "")

			if resp.StatusCode == 404 {
				//Post was not found
				if err := m.reportStorage.Delete(r.ID); err != nil {
					return fmt.Errorf("cannot remove outdated report: %w", err)
				}

				continue
			}

			if time.Now().After(r.LastReportUpdate.Add(30 * time.Second)) {
				if err := m.reportStorage.IncreaseCounter(r.ID); err != nil {
					return fmt.Errorf("cannot increase counter of report: %w", err)
				}

				if len(post.Attachments()) != 1 {
					return fmt.Errorf("got invalid post")
				}

				attachments := post.Attachments()

				for idx, f := range attachments[0].Fields {
					if f.Title == m.res.CountReportFromBot() {
						attachments[0].Fields[idx].Value = r.ReportTimes
					}
				}

				post.AddProp("attachments", attachments)

				if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
					return fmt.Errorf("cannot update post: %w", resp.Error)
				}
			}
		}
	}

	return nil
}

func (m *MattermostHandler) SendReport(objectID types.UID, namespaces, reason, resource, message, lastTimeStampStr string, count int32) error {
	var new *reportstorage.Report

	_, err := m.reportStorage.ReadByObjectID(objectID)

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
			IsInProgress:     false,
			LastReportUpdate: time.Now(),
			ReportStopped:    false,
			ReportStoppedBy:  "",
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
					Title: m.res.CountReportFromBot(),
					Value: new.ReportTimes,
					Short: true,
				},
			},
			Actions: []*model.PostAction{
				{
					Type:  "button",
					Name:  m.res.Submit(),
					Style: "success",
					Integration: &model.PostActionIntegration{
						//URL: fmt.Sprintf("http://localhost:8065/plugin/%s/report/submit?reportID=%v&user=%v", "net.mspielberger.k8s-bot-redirecter", new.ID.String(), "mspielberger"),
						URL: fmt.Sprintf("http://192.168.178.20:9090/report/submit?reportID=%v&user=%v", new.ID.String(), "test"),
					},
					Disabled: false,
				},
			},
		}}

		model.ParseSlackAttachment(post, attachment)

		if post, resp := m.client.CreatePost(post); resp.Error != nil {
			return fmt.Errorf("cannot create post for report: %w", resp.Error)
		} else {
			new.PostID = post.Id
			if err := m.reportStorage.Write(new); err != nil {
				return fmt.Errorf("cannot write report: %w", err)
			}
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("cannot read object: %w", err)
	}

	return nil
}

func (m *MattermostHandler) SubmitReport(reportID uuid2.UUID, username string) error {
	report, err := m.reportStorage.ReadByReportID(reportID)

	if err != nil {
		return fmt.Errorf("cannot read report with given id: %w", err)
	}

	post, resp := m.client.GetPost(report.PostID, "")

	if resp.Error != nil {
		return fmt.Errorf("cannot get post for report: %w", resp.Error)
	}

	if len(post.Attachments()) != 1 {
		return fmt.Errorf("got invalid post")
	}

	attachments := post.Attachments()

	if len(attachments[0].Actions) != 1 {
		return fmt.Errorf("got invalid post")
	}

	attachments[0].Fields = append(attachments[0].Fields, &model.SlackAttachmentField{
		Title: m.res.SubmittedAt(),
		Value: time.Now().Format("15:04:05 02.01.2006"),
		Short: true,
	})

	attachments[0].Fields = append(attachments[0].Fields, &model.SlackAttachmentField{
		Title: m.res.SubmittedBy(),
		Value: username,
		Short: true,
	})

	attachments[0].Actions[0].Disabled = true

	if err := m.reportStorage.SubmitReport(report.ID, username); err != nil {
		return fmt.Errorf("cannot update report in storage to submit: %w", err)
	}

	model.ParseSlackAttachment(post, attachments)

	if _, resp := m.client.UpdatePost(post.Id, post); resp.Error != nil {
		return fmt.Errorf("cannot update post: %w", resp.Error)
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
