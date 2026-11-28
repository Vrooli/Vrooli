package runs

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type experienceAgeReport struct {
	ScenarioName         string  `json:"scenarioName"`
	ExperienceFreshness  string  `json:"experienceFreshness"`
	LastExperienceStatus string  `json:"lastExperienceStatus,omitempty"`
	LastExperienceAt     *string `json:"lastExperienceAt,omitempty"`
	LastExperienceRunID  string  `json:"lastExperienceRunId,omitempty"`
	LastUIChangeAt       *string `json:"lastUIChangeAt,omitempty"`
}

// runExperienceAge reads the stored verdict summary. It never treats an
// absent or unrecognized freshness state as current.
func runExperienceAge(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs experience-age", flag.ContinueOnError)
	fs.SetOutput(w)
	scenario := fs.String("scenario", "", "Scenario slug")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 1 && strings.TrimSpace(*scenario) == "" {
		*scenario = fs.Args()[0]
	}
	scen, err := requireScenario(*scenario)
	if err != nil {
		return err
	}
	if apiClient == nil {
		return errors.New("test-genie API base URL is not configured")
	}
	body, err := apiClient.Get("/api/v1/scenarios/"+url.PathEscape(scen), nil)
	if err != nil {
		return err
	}
	var report experienceAgeReport
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("decode scenario verdict summary: %w", err)
	}
	if report.ScenarioName == "" {
		return errors.New("scenario summary response did not identify the scenario")
	}
	if err := validateExperienceAgeReport(report); err != nil {
		return &exitErr{code: exitNotComparable, err: err}
	}
	if *jsonOut {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(encoded))
	} else {
		fmt.Fprintf(w, "Scenario: %s\n", report.ScenarioName)
		fmt.Fprintf(w, "Experience verdict freshness: %s\n", formatExperienceFreshness(report.ExperienceFreshness))
		if report.LastExperienceAt == nil {
			fmt.Fprintln(w, "Stored verdict: never run")
		} else {
			fmt.Fprintf(w, "Stored verdict: %s · evaluated %s", displayStatus(report.LastExperienceStatus), *report.LastExperienceAt)
			if report.LastExperienceRunID != "" {
				fmt.Fprintf(w, " · run %s", report.LastExperienceRunID)
			}
			fmt.Fprintln(w)
		}
		if report.LastUIChangeAt != nil {
			fmt.Fprintf(w, "Latest UI source change: %s\n", *report.LastUIChangeAt)
		}
	}
	return experienceAgeExit(report.ExperienceFreshness, report.LastExperienceStatus)
}

func validateExperienceAgeReport(report experienceAgeReport) error {
	parseTime := func(label string, value *string) error {
		if value == nil {
			return fmt.Errorf("%s is missing", label)
		}
		if _, err := time.Parse(time.RFC3339Nano, *value); err != nil {
			return fmt.Errorf("%s is invalid: %w", label, err)
		}
		return nil
	}
	switch report.ExperienceFreshness {
	case "never_run":
		if report.LastExperienceAt != nil {
			return errors.New("never-run freshness conflicts with a stored verdict timestamp")
		}
	case "older_than_last_ui_change", "evaluated_after_last_ui_change":
		if err := parseTime("stored verdict timestamp", report.LastExperienceAt); err != nil {
			return err
		}
		if err := parseTime("latest UI source timestamp", report.LastUIChangeAt); err != nil {
			return err
		}
	case "ui_surface_unavailable", "ui_change_time_unavailable":
		if err := parseTime("stored verdict timestamp", report.LastExperienceAt); err != nil {
			return err
		}
	}
	return nil
}

func experienceAgeExit(freshness, status string) error {
	switch freshness {
	case "evaluated_after_last_ui_change":
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "passed":
			return nil
		case "failed":
			return &exitErr{code: exitRegression, err: errors.New("current experience verdict is failed")}
		default:
			return &exitErr{code: exitNotComparable, err: fmt.Errorf("current experience verdict is %q", status)}
		}
	case "never_run", "older_than_last_ui_change":
		return &exitErr{code: exitRegression, err: fmt.Errorf("experience verdict is %s", freshness)}
	default:
		return &exitErr{code: exitNotComparable, err: fmt.Errorf("experience verdict freshness is unavailable: %s", freshness)}
	}
}

func formatExperienceFreshness(freshness string) string {
	switch freshness {
	case "never_run":
		return "Never run"
	case "older_than_last_ui_change":
		return "Older than the last UI change"
	case "evaluated_after_last_ui_change":
		return "Evaluated after the last UI change"
	case "ui_surface_unavailable":
		return "No UI surface to compare"
	case "ui_change_time_unavailable":
		return "UI change time unavailable"
	default:
		return "Unknown freshness"
	}
}

func displayStatus(status string) string {
	status = strings.TrimSpace(strings.ReplaceAll(status, "_", " "))
	if status == "" {
		return "unknown result"
	}
	return strings.ToUpper(status[:1]) + status[1:]
}
