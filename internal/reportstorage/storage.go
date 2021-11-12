package reportstorage

import (
	"github.com/golangee/uuid"
	"k8s.io/apimachinery/pkg/types"
)

type ReportStorage interface {
	Write(report *Report) error
	ReadByReportID(reportID uuid.UUID) (*Report, error)
	ReadByObjectID(objectID types.UID) (*Report, error)
	Delete(reportID uuid.UUID) error

	IncreaseCounter(reportID uuid.UUID) error
	SetInProgress(reportID uuid.UUID, val bool) error

	SetPostID(reportID uuid.UUID, postID string) error
}
