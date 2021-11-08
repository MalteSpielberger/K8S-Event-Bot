package reportstorage

import "k8s.io/apimachinery/pkg/types"

type InMemoryReportStorage struct {
	reports map[types.UID]string
}

func NewInMemoryReportStorage() *InMemoryReportStorage {
	return &InMemoryReportStorage{
		reports: map[types.UID]string{},
	}
}

func (i *InMemoryReportStorage) Add(id types.UID, postId string) error {
	i.reports[id] = postId

	return nil
}

func (i *InMemoryReportStorage) GetPostId(id types.UID) (string, error) {
	return i.reports[id], nil
}

func (i *InMemoryReportStorage) WasReportedEarlier(id types.UID) (bool, error) {
	_, found := i.reports[id]

	return found, nil
}