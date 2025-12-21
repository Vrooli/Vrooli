package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ExecutionTimeline struct {
	Frames []TimelineFrame `json:"frames"`
}

type TimelineFrame struct {
	StepIndex         *int               `json:"step_index"`
	StepType          string             `json:"step_type"`
	Status            string             `json:"status"`
	Success           *bool              `json:"success"`
	DurationMs        *int               `json:"duration_ms"`
	TotalDurationMs   *int               `json:"total_duration_ms"`
	HighlightRegions  []any              `json:"highlight_regions"`
	MaskRegions       []any              `json:"mask_regions"`
	ConsoleLogCount   *int               `json:"console_log_count"`
	NetworkEventCount *int               `json:"network_event_count"`
	FocusedElement    *FocusedElementRef `json:"focused_element"`
	Screenshot        any                `json:"screenshot"`
	FinalURL          string             `json:"final_url"`
	Error             string             `json:"error"`
	Assertion         *TimelineAssertion `json:"assertion"`
	RetryAttempt      *int               `json:"retry_attempt"`
	RetryMaxAttempts  *int               `json:"retry_max_attempts"`
}

type FocusedElementRef struct {
	Selector string `json:"selector"`
}

type TimelineAssertion struct {
	Success  *bool  `json:"success"`
	Selector string `json:"selector"`
	Mode     string `json:"mode"`
	Message  string `json:"message"`
}

func SummarizeTimeline(data []byte) ([]string, error) {
	var timeline ExecutionTimeline
	if err := json.Unmarshal(data, &timeline); err != nil {
		return nil, err
	}

	frames := timeline.Frames
	sort.Slice(frames, func(i, j int) bool {
		left := 0
		right := 0
		if frames[i].StepIndex != nil {
			left = *frames[i].StepIndex
		}
		if frames[j].StepIndex != nil {
			right = *frames[j].StepIndex
		}
		return left < right
	})

	lines := make([]string, 0, len(frames))
	for _, frame := range frames {
		stepIndex := 0
		if frame.StepIndex != nil {
			stepIndex = *frame.StepIndex
		}
		label := frame.StepType
		if strings.TrimSpace(label) == "" {
			label = "step"
		}
		status := frame.Status
		if status == "" {
			status = "unknown"
		}
		success := true
		if frame.Success != nil {
			success = *frame.Success
		}
		duration := 0
		if frame.DurationMs != nil {
			duration = *frame.DurationMs
		}
		total := 0
		if frame.TotalDurationMs != nil {
			total = *frame.TotalDurationMs
		}
		highlights := len(frame.HighlightRegions)
		masks := len(frame.MaskRegions)
		consoleLogs := 0
		if frame.ConsoleLogCount != nil {
			consoleLogs = *frame.ConsoleLogCount
		}
		networkEvents := 0
		if frame.NetworkEventCount != nil {
			networkEvents = *frame.NetworkEventCount
		}

		line := fmt.Sprintf("%d. %s [%s] %s - %dms", stepIndex+1, label, status, successGlyph(success), duration)
		if total > 0 && total != duration {
			line += fmt.Sprintf(" (total %dms)", total)
		}
		line += fmt.Sprintf(" - console: %d network: %d", consoleLogs, networkEvents)

		if hasScreenshot(frame.Screenshot) {
			line += " - screenshot"
		}
		if frame.FocusedElement != nil && strings.TrimSpace(frame.FocusedElement.Selector) != "" {
			line += fmt.Sprintf(" - focus: %s", frame.FocusedElement.Selector)
		}
		if highlights > 0 {
			line += fmt.Sprintf(" - highlights: %d", highlights)
		}
		if masks > 0 {
			line += fmt.Sprintf(" - masks: %d", masks)
		}
		if strings.TrimSpace(frame.FinalURL) != "" {
			line += fmt.Sprintf(" - url: %s", frame.FinalURL)
		}
		if strings.TrimSpace(frame.Error) != "" {
			line += fmt.Sprintf(" - error: %s", frame.Error)
		}
		if frame.RetryAttempt != nil && *frame.RetryAttempt > 1 {
			line += fmt.Sprintf(" - retry: %d", *frame.RetryAttempt)
			if frame.RetryMaxAttempts != nil && *frame.RetryMaxAttempts > 0 {
				line += fmt.Sprintf("/%d", *frame.RetryMaxAttempts)
			}
		}
		if frame.Assertion != nil {
			label := strings.TrimSpace(frame.Assertion.Selector)
			if label == "" {
				label = strings.TrimSpace(frame.Assertion.Mode)
			}
			if label != "" {
				assertSuccess := true
				if frame.Assertion.Success != nil {
					assertSuccess = *frame.Assertion.Success
				}
				line += fmt.Sprintf(" - assertion: %s", label)
				if assertSuccess {
					line += " (ok)"
				} else {
					line += " (failed)"
					if strings.TrimSpace(frame.Assertion.Message) != "" {
						line += fmt.Sprintf(" - %s", frame.Assertion.Message)
					}
				}
			}
		}

		lines = append(lines, line)
	}

	return lines, nil
}

func successGlyph(ok bool) string {
	if ok {
		return "OK"
	}
	return "FAIL"
}

func hasScreenshot(value any) bool {
	if value == nil {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	default:
		return true
	}
}
