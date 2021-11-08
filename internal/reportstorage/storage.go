package reportstorage

import "k8s.io/apimachinery/pkg/types"

type ReportStorage interface {
	Add(id types.UID, postId string) error
	GetPostId(id types.UID) (string, error)
	WasReportedEarlier(id types.UID) (bool, error)
}