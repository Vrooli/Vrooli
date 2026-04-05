package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	baseURL := os.Getenv("VROOLI_EVENTS_URL")
	if baseURL == "" {
		port := os.Getenv("API_PORT")
		if port == "" {
			port = "8080"
		}
		baseURL = "http://localhost:" + port
	}

	switch os.Args[1] {
	case "query":
		cmdQuery(baseURL, os.Args[2:])
	case "subscribe":
		cmdSubscribe(baseURL, os.Args[2:])
	case "stats":
		cmdStats(baseURL, os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `vrooli-events — event bus CLI

Commands:
  query       Search events by type/source/correlation_id
  subscribe   Real-time SSE event listener
  stats       Show event store statistics

Use "vrooli-events <command> --help" for command-specific options.`)
}

func cmdQuery(baseURL string, args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario (exact)")
	corrID := fs.String("correlation-id", "", "correlation ID (exact)")
	since := fs.String("since", "", "return events after this ID")
	limit := fs.String("limit", "", "max results (default 100)")
	asJSON := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

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

	u := baseURL + "/api/v1/events"
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	resp, err := http.Get(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error (%d): %s\n", resp.StatusCode, body)
		os.Exit(1)
	}

	if *asJSON {
		fmt.Println(string(body))
		return
	}

	var events []map[string]any
	if err := json.Unmarshal(body, &events); err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	if len(events) == 0 {
		fmt.Println("No events found.")
		return
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
}

func cmdSubscribe(baseURL string, args []string) {
	fs := flag.NewFlagSet("subscribe", flag.ExitOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario pattern")
	target := fs.String("target", "", "target scenario pattern")
	asJSON := fs.Bool("json", false, "output raw SSE JSON data")
	_ = fs.Parse(args)

	params := url.Values{}
	if *typeFilter != "" {
		params.Set("type", *typeFilter)
	}
	if *source != "" {
		params.Set("source", *source)
	}
	if *target != "" {
		params.Set("target", *target)
	}

	u := baseURL + "/api/v1/events/subscribe"
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	resp, err := http.Get(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "error (%d): %s\n", resp.StatusCode, body)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Connected. Listening for events...")

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent, currentData string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, ":") {
			// Comment line (heartbeat)
			if !*asJSON {
				fmt.Println(line)
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			currentData = strings.TrimPrefix(line, "data: ")
			continue
		}
		if line == "" && currentEvent != "" {
			if *asJSON {
				fmt.Println(currentData)
			} else {
				fmt.Printf("[%s] %s\n", currentEvent, currentData)
			}
			currentEvent = ""
			currentData = ""
		}
	}
}

func cmdStats(baseURL string, args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if *asJSON {
		fmt.Println(string(body))
		return
	}

	var health map[string]any
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status:      %s\n", str(health, "status"))
	fmt.Printf("Subscribers: %v\n", health["subscribers"])
	if st, ok := health["store"].(map[string]any); ok {
		fmt.Printf("Events:      %v\n", st["totalEvents"])
		if bytes, ok := st["totalPayloadBytes"].(float64); ok {
			fmt.Printf("Store Size:  %.1f MB\n", bytes/1024/1024)
		}
	}
}

func str(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
