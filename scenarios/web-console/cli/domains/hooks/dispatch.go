package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

// dispatch is the portable hook entrypoint registered with Claude Code. It
// uses the installed web-console executable and the standard library only;
// no shell, jq, curl, or platform-specific script is required.
func dispatch(args []string) error {
	flags := flag.NewFlagSet("hooks dispatch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	event := flags.String("event", "", "Claude hook event")
	url := flags.String("url", "", "Web Console hook URL")
	token := flags.String("token", "", "Web Console hook token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *event == "" || *url == "" || *token == "" {
		return errors.New("hooks dispatch requires --event, --url, and --token")
	}
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read hook payload: %w", err)
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil
	}
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return fmt.Errorf("decode hook payload: %w", err)
	}
	if sessionID := os.Getenv("WC_WEB_CONSOLE_SESSION_ID"); sessionID != "" {
		document["web_console_session_id"] = sessionID
	}
	if *event == promptEvent {
		prompt, _ := document["prompt"].(string)
		if prompt == "" {
			return nil
		}
		document = map[string]any{"userPrompt": prompt, "webConsoleSessionId": os.Getenv("WC_WEB_CONSOLE_SESSION_ID")}
	}
	body, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode hook payload: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, *url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create hook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", *token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post hook payload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("hook endpoint returned %s", resp.Status)
	}
	return nil
}
