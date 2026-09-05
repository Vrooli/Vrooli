// fake-agent replays a recorded runner corpus for process-level tests.
// It intentionally has no network client and reads only FAKE_AGENT_CORPUS.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(run(os.Stdout, os.Stderr))
}

func run(stdout, stderr io.Writer) int {
	corpus := strings.TrimSpace(os.Getenv("FAKE_AGENT_CORPUS"))
	if corpus == "" {
		fmt.Fprintln(stderr, "FAKE_AGENT_CORPUS is required")
		return 2
	}
	// The marker is a test-observability hook. An unset value deliberately
	// disables it; replay itself is still fully controlled by the required
	// corpus above.
	// vrooli:env:optional
	if marker := strings.TrimSpace(os.Getenv("FAKE_AGENT_TAG_MARKER")); marker != "" {
		if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if err := os.WriteFile(marker, []byte(findTag()), 0o600); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}

	f, err := os.Open(corpus)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	defer f.Close()

	failure := false
	scanner := bufio.NewScanner(f)
	// Corpus lines can include full tool output. Keep the test agent's scanner
	// comfortably above the runner's normal JSON-event size.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(stdout, line)
		if strings.Contains(line, `"is_error":true`) || strings.Contains(line, `"success":false`) || strings.Contains(line, `"type":"error"`) {
			failure = true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if failure {
		return 1
	}
	return 0
}

func findTag() string {
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.HasSuffix(key, "AGENT_TAG") {
			return key + "=" + value
		}
	}
	return ""
}
