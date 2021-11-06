package errorsctx

import (
	"fmt"
	"k8s-event-bot/internal/k8s"
	"k8s-event-bot/internal/mattermost"
	"log"
	"time"
)

type ErrorListener struct {
	mattermost *mattermost.MattermostHandler
	api *k8s.KubernetesApi
}

func NewErrorsListener(handler *mattermost.MattermostHandler, api *k8s.KubernetesApi) *ErrorListener {
	return &ErrorListener{
		api: api,
		mattermost: handler,
	}
}

func (e *ErrorListener) Listen() error {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				log.Println("Send msg!")
				if err := e.mattermost.SendError("Hallo das ist eine Fehlermeldeun!"); err != nil {
					panic(err)
				}
			}
		}
	}()

	time.Sleep(15 * time.Second)
	ticker.Stop()
	done <- true
	fmt.Println("Ticker stopped")

	return nil
}


