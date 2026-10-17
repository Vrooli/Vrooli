package control

import "fmt"

type ResultItem struct {
	Name    string `json:"name"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type StartReport struct {
	Started []ResultItem `json:"started"`
	Failed  []ResultItem `json:"failed"`
	Message string       `json:"message"`
}

type StopReport struct {
	Stopped []ResultItem `json:"stopped"`
	Failed  []ResultItem `json:"failed"`
	Message string       `json:"message"`
}

func Started(name, message string) ResultItem {
	return ResultItem{Name: name, Message: message}
}

func Stopped(name, message string) ResultItem {
	return ResultItem{Name: name, Message: message}
}

func Failed(name string, err error) ResultItem {
	item := ResultItem{Name: name}
	if err != nil {
		item.Error = err.Error()
	}
	return item
}

func StartSummary(started, failed int) string {
	return fmt.Sprintf("Started %d targets, %d failed", started, failed)
}

func StopSummary(stopped, failed int) string {
	return fmt.Sprintf("Stopped %d targets, %d failed", stopped, failed)
}
