package pvcctx

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8sbot/internal/k8s"
	"k8sbot/internal/mattermost"
	"log"
	"time"
)

type EventListener struct {
	mattermost            *mattermost.MattermostHandler
	api                   *k8s.KubernetesApi
	warnOnPercentageUsage int
}

func NewEventListener(handler *mattermost.MattermostHandler, api *k8s.KubernetesApi, warnOnPercentageUsage int) *EventListener {
	return &EventListener{
		mattermost: handler,
		api: api,
		warnOnPercentageUsage: warnOnPercentageUsage,
	}
}

func (e *EventListener) check() error {
	namespaceList, err := e.api.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})

	if err != nil {
		return fmt.Errorf("cannot get namespaces: %w", err)
	}

	for _, namespace := range namespaceList.Items {
		pvcList, err := e.api.CoreV1().PersistentVolumeClaims(namespace.GetName()).List(context.Background(), v1.ListOptions{})

		if err != nil {
			return fmt.Errorf("cannot get pvcs from namespace %v: %w", namespace.GetName(), err)
		}

		for _, pvc := range pvcList.Items {
			log.Println("PVC: ", pvc.GetName())
		}
	}

	return nil
}

func (e *EventListener) Listen(done <-chan bool) error {
	ticket := time.NewTicker(10 * time.Second)

	go func() {
		for {
			select {
			case <-done:
				ticket.Stop()
				return
			case _ = <-ticket.C:
				if err := e.check(); err != nil {
					e.mattermost.SendInternalError(err)
				}
			}
		}
	}()

	return nil
}
