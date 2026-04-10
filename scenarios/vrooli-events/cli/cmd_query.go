package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
)

func (a *App) cmdQuery(args []string) error {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario (exact)")
	corrID := fs.String("correlation-id", "", "correlation ID (exact)")
	since := fs.String("since", "", "return events after this ID")
	limit := fs.String("limit", "", "max results (default 100)")
	asJSON := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	params := url.Values{}
	if *typeFilter != "" {
		params.Set("type", *typeFilter)
	}
	if *source != "" {
		params.Set("source", *source)
	}
	if *corrID != "" {
		params.Set("correlation_id", *corrID)
	}
	if *since != "" {
		params.Set("since", *since)
	}
	if *limit != "" {
		params.Set("limit", *limit)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/events"), params)
	if err != nil {
		return err
	}

	if *asJSON {
		fmt.Println(string(body))
		return nil
	}

	var events []map[string]any
	if err := json.Unmarshal(body, &events); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No events found.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "EVENT ID\tTYPE\tSOURCE\tTARGET\tCORRELATION")
	for _, e := range events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			str(e, "eventId"),
			str(e, "eventType"),
			str(e, "sourceScenario"),
			str(e, "targetScenario"),
			str(e, "correlationId"),
		)
	}
	tw.Flush()
	return nil
}

func str(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
