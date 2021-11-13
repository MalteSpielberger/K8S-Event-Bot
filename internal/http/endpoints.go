package http

import (
	"github.com/golangee/uuid"
	"k8sbot/internal/mattermost"
	"log"
	"net/http"
)

type ReportEndpoints struct {
	handler *mattermost.MattermostHandler
}

func NewReportEndpoints(handler *mattermost.MattermostHandler) *ReportEndpoints {
	r := &ReportEndpoints{
		handler: handler,
	}

	http.HandleFunc("/report/submit", r.handleSubmit)

	return r
}

func (r *ReportEndpoints) handleSubmit(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		log.Println("cannot parse form from user: ", err.Error())
		return
	}

	idStr := request.FormValue("reportID")

	id, err := uuid.Parse(idStr)

	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		log.Println("cannot parse given id: ", err)
	}

	username := request.FormValue("user")

	if err := r.handler.SubmitReport(id, username); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Println("cannot submit report: ", err.Error())
		return
	}

	writer.WriteHeader(http.StatusOK)
}
