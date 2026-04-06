package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdIngest(args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	eventID := fs.String("event-id", "", "unique event ID (required)")
	eventType := fs.String("type", "", "event type, e.g. app.user.created.v1 (required)")
	source := fs.String("source", "", "source scenario name (required)")
	target := fs.String("target", "", "target scenario name")
	corrID := fs.String("correlation-id", "", "correlation ID for tracing")
	payload := fs.String("payload", "", "JSON payload string or @file.json")
	asJSON := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *eventID == "" || *eventType == "" || *source == "" {
		return fmt.Errorf("usage: ingest --event-id ID --type TYPE --source SOURCE [--target TARGET] [--correlation-id CID] [--payload JSON] [--json]")
	}

	envelope := map[string]any{
		"eventId":        *eventID,
		"eventType":      *eventType,
		"sourceScenario": *source,
	}
	if *target != "" {
		envelope["targetScenario"] = *target
	}
	if *corrID != "" {
		envelope["correlationId"] = *corrID
	}
	if *payload != "" {
		payloadStr := *payload
		// Support @file.json syntax
		if len(payloadStr) > 0 && payloadStr[0] == '@' {
			f, err := os.Open(payloadStr[1:])
			if err != nil {
				return fmt.Errorf("read payload file: %w", err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("read payload file: %w", err)
			}
			payloadStr = string(data)
		}
		var parsed any
		if err := json.Unmarshal([]byte(payloadStr), &parsed); err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
		envelope["payload"] = payloadStr
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/events"), nil, envelope)
	if err != nil {
		return err
	}

	if *asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if resp["dry_run"] == true {
		fmt.Printf("Dry run: event %s would be accepted (type: %s, source: %s)\n",
			*eventID, *eventType, *source)
		return nil
	}

	fmt.Printf("Ingested event: %s (store ID: %v)\n", *eventID, resp["id"])
	return nil
}
