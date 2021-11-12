package reportstorage

type NoReportErr struct {

}

func (n *NoReportErr) Error() string {
	return "no report found"
}
