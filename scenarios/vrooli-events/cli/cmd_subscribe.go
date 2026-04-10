package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (a *App) cmdSubscribe(args []string) error {
	fs := flag.NewFlagSet("subscribe", flag.ExitOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario pattern")
	target := fs.String("target", "", "target scenario pattern")
	asJSON := fs.Bool("json", false, "output raw SSE JSON data")
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
	if *target != "" {
		params.Set("target", *target)
	}

	u := a.core.APIClient.BaseURL() + a.apiPath("/events/subscribe")
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error (%d): %s", resp.StatusCode, body)
	}

	fmt.Fprintln(os.Stderr, "Connected. Listening for events...")

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent, currentData string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, ":") {
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
	return nil
}
