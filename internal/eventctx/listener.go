package eventctx

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8sbot/internal/k8s"
	"k8sbot/internal/mattermost"
	"time"
)

type EventListener struct {
	mattermost         *mattermost.MattermostHandler
	api                *k8s.KubernetesApi
	warnOnEventReasons []string
	count              int
}

func NewEventListener(handler *mattermost.MattermostHandler, api *k8s.KubernetesApi, warnOnEventReasons []string, count int) *EventListener {
	return &EventListener{
		api:                api,
		mattermost:         handler,
		warnOnEventReasons: warnOnEventReasons,
		count:              count,
	}
}

func (e *EventListener) check() error {
	namespaceList, err := e.api.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})

	if err != nil {
		return fmt.Errorf("cannot get namespaces: %w", err)
	}

	for _, namespace := range namespaceList.Items {
		eventList, err := e.api.CoreV1().Events(namespace.GetName()).List(context.Background(), v1.ListOptions{})

		if err != nil {
			return fmt.Errorf("cannot get event-list for namespace %v: %w", namespace.GetName(), err)
		}

		for _, event := range eventList.Items {
			if event.Type == "Warning" {
				for _, reason := range e.warnOnEventReasons {
					if reason == event.Reason {
						if event.Count >= int32(e.count) {
							if err := e.mattermost.SendEventWarning(event.ObjectMeta.UID, event.Namespace, event.Reason, event.ObjectMeta.Name, event.Message, event.LastTimestamp.String(), event.Count); err != nil {
								return fmt.Errorf("cannot send event warning: %w", err)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (e *EventListener) Listen(done <-chan bool) error {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case _ = <-ticker.C:
				if err := e.check(); err != nil {
					e.mattermost.SendInternalError(err)
				}
			}
		}
	}()

	return nil
}
