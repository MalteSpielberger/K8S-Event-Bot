package reportstorage

import (
	"github.com/golangee/uuid"
	"k8s.io/apimachinery/pkg/types"
)

type ReportStorage interface {
	ReadAll() ([]*Report, error)
	Write(report *Report) error
	ReadByReportID(reportID uuid.UUID) (*Report, error)
	ReadByObjectID(objectID types.UID) (*Report, error)
	Delete(reportID uuid.UUID) error

	IncreaseCounter(reportID uuid.UUID) error
	SetInProgress(reportID uuid.UUID, val bool) error
	SubmitReport(reportID uuid.UUID, username string) error

	SetPostID(reportID uuid.UUID, postID string) error
}
