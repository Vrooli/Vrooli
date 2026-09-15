package credentials

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Scanner finds synthetic credential canaries in retained surfaces. A canary
// is a value minted for a test and never legitimately retained; any hit in a
// ledger dump, a log, an argv capture, a receipt or an evidence file is a
// leak. Surfaces are named so a finding says where the value was found.
type Scanner struct {
	mu       sync.Mutex
	canaries []string
	surfaces map[string][]byte
}

// NewScanner returns a scanner for the given canary values.
func NewScanner(canaries ...string) *Scanner {
	s := &Scanner{surfaces: map[string][]byte{}}
	s.Add(canaries...)
	return s
}

// Add registers canary values; empty strings are ignored.
func (s *Scanner) Add(canaries ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range canaries {
		if strings.TrimSpace(c) != "" {
			s.canaries = append(s.canaries, c)
		}
	}
}

// Canaries lists the registered values.
func (s *Scanner) Canaries() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.canaries...)
}

// Record retains one surface's bytes under a name; a repeated name appends.
func (s *Scanner) Record(surface string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.surfaces[surface] = append(s.surfaces[surface], data...)
	s.surfaces[surface] = append(s.surfaces[surface], '\n')
}

// RecordString retains a string surface.
func (s *Scanner) RecordString(surface, text string) { s.Record(surface, []byte(text)) }

// RecordJSON retains any value by its JSON encoding, so a map or struct that
// would be persisted or returned to a client is scanned as it would be seen.
func (s *Scanner) RecordJSON(surface string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.Record(surface, data)
	return nil
}

// Finding is one canary present in one surface.
type Finding struct {
	Surface string `json:"surface"`
	Canary  string `json:"canary"`
	Offset  int    `json:"offset"`
}

func (f Finding) String() string {
	return fmt.Sprintf("canary %q found in surface %q at offset %d", redact(f.Canary), f.Surface, f.Offset)
}

// Scan reports every canary occurrence in every recorded surface, sorted by
// surface then canary. An empty result is the leakage-free verdict.
func (s *Scanner) Scan() []Finding {
	s.mu.Lock()
	defer s.mu.Unlock()
	var findings []Finding
	for surface, data := range s.surfaces {
		for _, canary := range s.canaries {
			if idx := strings.Index(string(data), canary); idx >= 0 {
				findings = append(findings, Finding{Surface: surface, Canary: canary, Offset: idx})
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Surface != findings[j].Surface {
			return findings[i].Surface < findings[j].Surface
		}
		return findings[i].Canary < findings[j].Canary
	})
	return findings
}

// Surfaces lists the recorded surface names.
func (s *Scanner) Surfaces() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.surfaces))
	for name := range s.surfaces {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// redact keeps a finding message useful without repeating the value.
func redact(canary string) string {
	if len(canary) <= 8 {
		return strings.Repeat("*", len(canary))
	}
	return canary[:4] + strings.Repeat("*", len(canary)-8) + canary[len(canary)-4:]
}
