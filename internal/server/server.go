package server

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"k8sbot/internal/configuration"
	"k8sbot/internal/eventctx"
	"k8sbot/internal/http"
	"k8sbot/internal/k8s"
	"k8sbot/internal/listener"
	"k8sbot/internal/mattermost"
	"k8sbot/internal/reportstorage"
	http2 "net/http"
)

type Server struct {
	config            *configuration.Configuration
	endpoints         *http.ReportEndpoints
	mattermostHandler *mattermost.MattermostHandler
	k8sApi            *k8s.KubernetesApi
	client            *model.Client4
	botUser           *model.User
	listeners         []listener.Listener
}

func NewServer() *Server {
	return &Server{
		config: configuration.NewConfiguration(),
	}
}

func (s *Server) getReportEndpoints() (*http.ReportEndpoints, error) {
	if s.endpoints == nil {
		handler, err := s.getMattermostHandler()

		if err != nil {
			return nil, err
		}

		s.endpoints = http.NewReportEndpoints(handler)
	}

	return s.endpoints, nil
}

func (s *Server) getMattermostHandler() (*mattermost.MattermostHandler, error) {
	if s.mattermostHandler == nil {
		user, err := s.getBotUser()

		if err != nil {
			return nil, fmt.Errorf("cannot get bot user: %w", err)
		}

		s.mattermostHandler = mattermost.NewMattermostHandler(s.config.BotWantedUsername, user, s.getMattermostClient(), s.config.MaintainerUsernames, s.config.DevOpsChannel, s.config.TeamID, reportstorage.NewInMemoryReportStorage())
	}

	return s.mattermostHandler, nil
}

func (s *Server) getMattermostClient() *model.Client4 {
	if s.client == nil {
		s.client = model.NewAPIv4Client(s.config.MattermostHost)
	}

	return s.client
}

func (s *Server) getBotUser() (*model.User, error) {
	if s.botUser == nil {
		user, resp := s.getMattermostClient().Login(s.config.BotUsername, s.config.BotPassword)

		if resp.Error != nil {
			return nil, fmt.Errorf("cannot login to user: %w", resp.Error)
		}

		s.botUser = user
	}

	return s.botUser, nil
}

func (s *Server) getKubernetesApi() (*k8s.KubernetesApi, error) {
	if s.k8sApi == nil {
		api, err := k8s.NewKubernetesApi()

		if err != nil {
			return nil, err
		}

		s.k8sApi = api
	}

	return s.k8sApi, nil
}

func (s *Server) getListeners() ([]listener.Listener, error) {
	if s.listeners == nil {
		s.listeners = []listener.Listener{}

		handler, err := s.getMattermostHandler()

		if err != nil {
			return nil, fmt.Errorf("cannot get mattermost handler: %w", err)
		}

		k8sApi, err := s.getKubernetesApi()

		if err != nil {
			return nil, fmt.Errorf("cannot get kubernetes api: %w", err)
		}

		s.listeners = append(s.listeners, eventctx.NewEventListener(handler, k8sApi, s.config.WarnOnEventReasons, s.config.WarnOnReachCount))
	}

	return s.listeners, nil
}

func (s *Server) Start() error {
	// Init the bot
	if _, err := s.getBotUser(); err != nil {
		return fmt.Errorf("init bot failed: %w", err)
	}

	if _, err := s.getKubernetesApi(); err != nil {
		return fmt.Errorf("inti k8s-api failed: %w", err)
	}

	listeners, err := s.getListeners()

	if err != nil {
		return fmt.Errorf("cannot get listeners: %w", err)
	}

	done := make(chan bool)

	for i, l := range listeners {
		if err := l.Listen(done); err != nil {
			return fmt.Errorf("cannot start l %v: %w", i, err)
		}
	}

	handler, err := s.getMattermostHandler()

	if err != nil {
		return err
	}

	if err := handler.Listen(done); err != nil {
		return fmt.Errorf("cannot check reports: %w", err)
	}

	_, err = s.getReportEndpoints()

	if err != nil {
		return err
	}

	if err := http2.ListenAndServe(":9090", nil); err != nil {
		panic(err)
	}

	return nil
}
