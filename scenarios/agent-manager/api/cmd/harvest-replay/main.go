// harvest-replay builds a small, redacted codec transcript corpus from durable
// agent-manager run directories. It performs no network or process launch.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type meta struct {
	RunnerType string `json:"runner_type"`
}

var sensitive = []*regexp.Regexp{
	regexp.MustCompile(`(?i)("(?:api[_-]?key|token|secret|authorization|access_token|refresh_token)"\s*:\s*)"[^"]*"`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|authorization|access_token|refresh_token)\s*=\s*)[^\s]+`),
	regexp.MustCompile(`/home/[^/"\\s]+`),
}

func main() {
	root := flag.String("root", "", "agent-manager runs directory")
	out := flag.String("out", "internal/adapters/runner/codecs/testdata/corpus", "output corpus directory")
	max := flag.Int("max-per-runner", 25, "maximum transcripts per runner")
	flag.Parse()
	if *root == "" || *max < 1 {
		fmt.Fprintln(os.Stderr, "--root and a positive --max-per-runner are required")
		os.Exit(2)
	}
	entries, err := os.ReadDir(*root)
	if err != nil {
		fail(err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	counts := map[string]int{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(*root, entry.Name())
		data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
		if err != nil {
			continue
		}
		var m meta
		if json.Unmarshal(data, &m) != nil || m.RunnerType == "" || counts[m.RunnerType] >= *max {
			continue
		}
		transcript, err := os.ReadFile(filepath.Join(dir, "transcript.ndjson"))
		if err != nil || len(strings.TrimSpace(string(transcript))) == 0 {
			continue
		}
		redacted := redact(string(transcript))
		if err := os.MkdirAll(*out, 0o755); err != nil {
			fail(err)
		}
		name := fmt.Sprintf("%s-%02d.jsonl", m.RunnerType, counts[m.RunnerType]+1)
		if err := os.WriteFile(filepath.Join(*out, name), []byte(redacted), 0o600); err != nil {
			fail(err)
		}
		counts[m.RunnerType]++
	}
	for runner, count := range counts {
		fmt.Printf("%s: %d\n", runner, count)
	}
}

func redact(value string) string {
	value = sensitive[0].ReplaceAllString(value, `${1}"<REDACTED>"`)
	value = sensitive[1].ReplaceAllString(value, `${1}<REDACTED>`)
	return sensitive[2].ReplaceAllString(value, "<HOME>")
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
