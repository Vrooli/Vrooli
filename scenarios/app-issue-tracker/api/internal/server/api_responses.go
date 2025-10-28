package server

type IssueListData struct {
	Issues []Issue `json:"issues"`
	Count  int     `json:"count"`
}

type IssueCreateData struct {
	Issue       *Issue `json:"issue"`
	IssueID     string `json:"issue_id"`
	StoragePath string `json:"storage_path"`
}

type IssueData struct {
	Issue *Issue `json:"issue"`
}

type IssueDeleteData struct {
	IssueID string `json:"issue_id"`
}

type IssueSearchData struct {
	Results []Issue `json:"results"`
	Count   int     `json:"count"`
	Query   string  `json:"query"`
	Method  string  `json:"method"`
}

type ProcessorSummary struct {
	Processor           ProcessorState `json:"processor"`
	IssuesProcessed     int            `json:"issues_processed"`
	IssuesRemainingMode string         `json:"issues_remaining_mode"`
	IssuesRemaining     *int           `json:"issues_remaining_count,omitempty"`
}

type RunningProcessesData struct {
	Processes []*RunningProcess `json:"processes"`
}

type StopProcessData struct {
	IssueID string `json:"issue_id"`
}

type ProcessorCounterResetData struct {
	IssuesProcessed int `json:"issues_processed"`
}

type ProcessorUpdatedData struct {
	Processor ProcessorState `json:"processor"`
}
