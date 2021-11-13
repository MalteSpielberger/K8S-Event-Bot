package reportstorage

import (
	"github.com/golangee/uuid"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

type Report struct {
	ID               uuid.UUID
	ReportedObject   types.UID // ID of the k8s object
	PostID           string
	Namespace        string
	Reason           string
	Resource         string
	Msg              string
	Count            int32 // Num how often the issue happens in the cluster
	ReportTimes      int   // How often the same issue was reported
	IsInProgress     bool  // Is used, to check if a maintainer checks the issue
	LastReportUpdate time.Time
	ReportStopped    bool
	ReportStoppedBy  string
}
