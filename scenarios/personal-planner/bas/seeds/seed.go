package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	seedModeEnv = "TEST_GENIE_SEEDS"
	testModeEnv = "PERSONAL_PLANNER_TEST_MODE"
)

type event struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Subject          string `json:"subject"`
	Availability     string `json:"availability"`
	AllDay           bool   `json:"all_day"`
	StartDate        string `json:"start_date"`
	EndDateExclusive string `json:"end_date_exclusive"`
	StartAt          string `json:"start_at"`
	EndAt            string `json:"end_at"`
	Timezone         string `json:"timezone"`
	Revision         int64  `json:"revision"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv(seedModeEnv) != "1" {
		return fmt.Errorf("refusing to run outside the Test Genie seed lifecycle")
	}
	testRoot := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_ROOT"))
	dbPath := strings.TrimSpace(os.Getenv("PLAYBOOKS_SQLITE_PATH"))
	if testRoot == "" || dbPath == "" {
		return fmt.Errorf("isolated Test Genie SQLite root is required")
	}
	root, err := filepath.Abs(testRoot)
	if err != nil {
		return fmt.Errorf("resolve test storage root: %w", err)
	}
	database, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("resolve test database path: %w", err)
	}
	relative, err := filepath.Rel(root, database)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("database path is outside the isolated Test Genie storage root")
	}
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("TEST_GENIE_TARGET_API_BASE")), "/")
	if apiBase == "" {
		return fmt.Errorf("Test Genie target API base is required")
	}
	repoRoot := strings.TrimSpace(os.Getenv("TEST_GENIE_REPO_ROOT"))
	scenarioDir := strings.TrimSpace(os.Getenv("TEST_GENIE_SCENARIO_DIR"))
	cliDir := filepath.Join(scenarioDir, "cli")
	if repoRoot == "" || scenarioDir == "" {
		return fmt.Errorf("Test Genie scenario and repository roots are required")
	}
	info, err := os.Stat(filepath.Join(cliDir, "go.mod"))
	if err != nil {
		return fmt.Errorf("Personal Planner CLI module is unavailable: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("Personal Planner CLI module path is a directory")
	}

	key := fmt.Sprintf("e1-isolated-cli-%d", time.Now().UTC().UnixNano())
	createOutput, err := runCLI(cliDir, apiBase,
		"calendar", "create-event",
		"--title", "E1 isolated CLI synthetic",
		"--subject", "Q1 synthetic",
		"--notes", "Test Genie isolated test-pool record",
		"--availability", "busy",
		"--timezone", "America/New_York",
		"--all-day", "true",
		"--start-date", "2099-10-30",
		"--end-date-exclusive", "2099-11-03",
		"--idempotency-key", key,
	)
	if err != nil {
		return fmt.Errorf("supported CLI create on isolated test storage: %w", err)
	}
	createdID := capture(`Saved event ([[:alnum:]_-]+) at revision 1`, createOutput)
	if createdID == "" {
		return fmt.Errorf("CLI create did not report a durable event id and revision 1")
	}
	if _, err := runCLI(cliDir, apiBase, "calendar", "event-status", "--idempotency-key", key); err != nil {
		return fmt.Errorf("supported CLI idempotency lookup: %w", err)
	}
	if _, err := runCLI(cliDir, apiBase, "calendar", "get-event", "--event-id", createdID); err != nil {
		return fmt.Errorf("supported CLI get after create: %w", err)
	}
	if _, err := runCLI(cliDir, apiBase,
		"calendar", "update-event",
		"--event-id", createdID,
		"--revision", "1",
		"--title", "E1 isolated CLI revised",
		"--subject", "Q1 free awareness",
		"--notes", "Edited through the supported CLI",
		"--availability", "free",
		"--timezone", "America/New_York",
		"--all-day", "true",
		"--start-date", "2099-10-30",
		"--end-date-exclusive", "2099-11-03",
	); err != nil {
		return fmt.Errorf("supported CLI revision-guarded update: %w", err)
	}
	if _, err := runCLI(cliDir, apiBase, "calendar", "events", "--start-date", "2099-10-30", "--end-date", "2099-11-03"); err != nil {
		return fmt.Errorf("supported CLI date-range readback: %w", err)
	}
	statusOutput, err := runCLI(cliDir, apiBase, "calendar", "event-status", "--idempotency-key", key)
	if err != nil {
		return fmt.Errorf("supported CLI status read after update: %w", err)
	}
	if !strings.Contains(statusOutput, createdID) || !strings.Contains(statusOutput, "revision 2") {
		return fmt.Errorf("CLI status read did not preserve event identity and revision 2")
	}
	if err := verifyIndependentAPIRead(apiBase, createdID); err != nil {
		return err
	}

	timedKey := key + "-dst-fold"
	timedCreateOutput, err := runCLI(cliDir, apiBase,
		"calendar", "create-event",
		"--title", "E1 isolated CLI DST fold",
		"--subject", "owner",
		"--notes", "Synthetic New York fall-back event",
		"--availability", "busy",
		"--timezone", "America/New_York",
		"--all-day", "false",
		"--start-at", "2026-11-01T01:30:00-04:00",
		"--end-at", "2026-11-01T01:30:00-05:00",
		"--idempotency-key", timedKey,
	)
	if err != nil {
		return fmt.Errorf("supported CLI timed DST-fold create on isolated test storage: %w", err)
	}
	timedID := capture(`Saved event ([[:alnum:]_-]+) at revision 1`, timedCreateOutput)
	if timedID == "" {
		return fmt.Errorf("CLI timed create did not report a durable event id and revision 1")
	}
	if _, err := runCLI(cliDir, apiBase, "calendar", "get-event", "--event-id", timedID); err != nil {
		return fmt.Errorf("supported CLI timed event reopen: %w", err)
	}
	if _, err := runCLI(cliDir, apiBase, "calendar", "events", "--start-date", "2026-11-01", "--end-date", "2026-11-01"); err != nil {
		return fmt.Errorf("supported CLI timed event date-range readback: %w", err)
	}
	if err := verifyIndependentTimedAPIRead(apiBase, timedID); err != nil {
		return err
	}
	fmt.Println("isolated CLI/API synthetic all-day and DST-fold event create, idempotency lookup, revisioned edit, list, reopen and independent API readback passed")
	return nil
}

func verifyIndependentTimedAPIRead(apiBase, eventID string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	requestURL := strings.TrimRight(apiBase, "/") + "/api/v1/calendar/events/" + url.PathEscape(eventID)
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create independent timed API readback request: %w", err)
	}
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("independent timed API readback: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("independent timed API readback returned %s", response.Status)
	}
	var saved event
	if err := json.NewDecoder(response.Body).Decode(&saved); err != nil {
		return fmt.Errorf("decode independent timed API readback: %w", err)
	}
	if saved.ID != eventID || saved.Title != "E1 isolated CLI DST fold" || saved.Timezone != "America/New_York" || saved.AllDay || saved.StartAt != "2026-11-01T01:30:00-04:00" || saved.EndAt != "2026-11-01T01:30:00-05:00" || saved.Revision != 1 {
		return fmt.Errorf("independent timed API readback did not retain the expected New York DST-fold instants")
	}
	return nil
}

func runCLI(directory, apiBase string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	commandArgs := append([]string{"run", "."}, args...)
	cmd := exec.CommandContext(ctx, "go", commandArgs...)
	cmd.Dir = directory
	env := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "API_BASE_URL=") || strings.HasPrefix(entry, testModeEnv+"=") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "API_BASE_URL="+apiBase, testModeEnv+"=1")
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("go run . %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func capture(pattern, value string) string {
	match := regexp.MustCompile(pattern).FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func verifyIndependentAPIRead(apiBase, eventID string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	requestURL := strings.TrimRight(apiBase, "/") + "/api/v1/calendar/events/" + url.PathEscape(eventID)
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create independent API readback request: %w", err)
	}
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("independent API readback: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("independent API readback returned %s", response.Status)
	}
	var saved event
	if err := json.NewDecoder(response.Body).Decode(&saved); err != nil {
		return fmt.Errorf("decode independent API readback: %w", err)
	}
	if saved.ID != eventID || saved.Title != "E1 isolated CLI revised" || saved.Subject != "Q1 free awareness" || saved.Availability != "free" || !saved.AllDay || saved.StartDate != "2099-10-30" || saved.EndDateExclusive != "2099-11-03" || saved.Revision != 2 {
		return fmt.Errorf("independent API readback did not retain the expected synthetic event identity, revision, availability and exclusive civil dates")
	}
	return nil
}
