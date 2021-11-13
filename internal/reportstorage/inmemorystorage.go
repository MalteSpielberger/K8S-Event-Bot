package reportstorage

import (
	"github.com/golangee/uuid"
	"k8s.io/apimachinery/pkg/types"
)

type InMemoryReportStorage struct {
	reports []*Report
}

func NewInMemoryReportStorage() *InMemoryReportStorage {
	return &InMemoryReportStorage{
		reports: []*Report{},
	}
}

func (i *InMemoryReportStorage) ReadAll() ([]*Report, error) {
	return i.reports, nil
}

func (i *InMemoryReportStorage) Write(report *Report) error {
	i.reports = append(i.reports, report)

	return nil
}

func (i *InMemoryReportStorage) ReadByReportID(reportID uuid.UUID) (*Report, error) {
	for _, r := range i.reports {
		if r.ID == reportID {
			return r, nil
		}
	}

	return nil, &NoReportErr{}
}

func (i *InMemoryReportStorage) ReadByObjectID(objectID types.UID) (*Report, error) {
	for _, r := range i.reports {
		if r.ReportedObject == objectID {
			return r, nil
		}
	}

	return nil, &NoReportErr{}
}

func (i *InMemoryReportStorage) Delete(reportID uuid.UUID) error {
	tmp := []*Report{}

	for _, r := range i.reports {
		if r.ID != reportID {
			tmp = append(tmp, r)
		}
	}

	i.reports = tmp

	return nil
}


func (i *InMemoryReportStorage) IncreaseCounter(reportID uuid.UUID) error {
	for _, r := range i.reports {
		if r.ID == reportID {
			r.ReportTimes++
		}
	}

	return nil
}

func (i *InMemoryReportStorage) SetInProgress(reportID uuid.UUID, val bool) error {
	for _, r := range i.reports {
		if r.ID == reportID {
			r.IsInProgress = val
		}
	}

	return nil
}

func (i *InMemoryReportStorage) SubmitReport(reportID uuid.UUID, username string) error {
	for _, r := range i.reports {
		if r.ID == reportID {
			r.ReportStopped = true
			r.ReportStoppedBy = username
		}
	}

	return nil
}

func (i *InMemoryReportStorage) SetPostID(reportID uuid.UUID, postID string) error {
	for _, r := range i.reports {
		if r.ID == reportID {
			r.PostID = postID
		}
	}

	return nil
}
