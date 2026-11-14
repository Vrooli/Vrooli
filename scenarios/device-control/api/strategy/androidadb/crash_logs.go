package androidadb

import (
	"regexp"
	"strings"
)

// packageJavaCrashes selects exact Process identity from the bounded Android
// crash buffer. Ordinary logs from other apps are never returned as a fallback.
func packageJavaCrashes(data, pkg string) []byte {
	linePattern := regexp.MustCompile(`^[A-Z]/AndroidRuntime\(\s*([0-9]+)\):\s?(.*)$`)
	processPattern := regexp.MustCompile(`^Process:\s*` + regexp.QuoteMeta(pkg) + `,\s*PID:\s*[0-9]+\b`)
	type block struct {
		lines   []string
		matched bool
	}
	active := map[string]*block{}
	var completed []*block
	for _, line := range strings.Split(data, "\n") {
		match := linePattern.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		pid, body := match[1], match[2]
		if strings.HasPrefix(body, "FATAL EXCEPTION:") {
			b := &block{}
			active[pid] = b
			completed = append(completed, b)
		}
		b := active[pid]
		if b == nil {
			continue
		}
		if len(b.lines) < 80 {
			b.lines = append(b.lines, line)
		}
		b.matched = b.matched || processPattern.MatchString(body)
	}
	var out []string
	for _, b := range completed {
		if b.matched {
			out = append(out, b.lines...)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return []byte(strings.Join(out, "\n") + "\n")
}
