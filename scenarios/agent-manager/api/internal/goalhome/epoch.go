package goalhome

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// Epoch is the parsed brief, exit gate, directives and slice log of one
// epochs/E<n>.md file (large-effort-orchestration §4.2).
type Epoch struct {
	Fields      map[string]string
	Directives  []Directive
	Log         []SliceLine
	Acceptances []Acceptance
	Malformed   []string
	Gates       []Gate
	Amendments  []Amendment
	// GateNotes are the exit-gate and amendment lines that are not typed
	// items: the whole legacy gate, or prose around typed items.
	GateNotes []Finding
	// HasExitGate is true when the file has a "## Exit gate" section.
	HasExitGate bool
}

// Directive is one "- D<n> <time> <text>" line under ## Directives.
type Directive struct {
	ID   string    `json:"id"`
	At   time.Time `json:"at,omitempty"`
	Text string    `json:"text"`
}

// SliceLine is one work unit:
// <ISO time> | <what changed> | exit metric=<v> | net runtime lines=<±n> | net test lines=<±n> | ack=<text naming D<n> ids, or ->
// Optional cells "gate G<n>=pass|fail|unverified [value]" record gate results.
type SliceLine struct {
	At           time.Time  `json:"at"`
	What         string     `json:"what"`
	Metric       string     `json:"metric"`
	RuntimeLines int        `json:"net_runtime_lines"`
	TestLines    int        `json:"net_test_lines"`
	Acks         []string   `json:"acks"`
	Gates        []GateCell `json:"gates,omitempty"`
}

// GateCell is one recorded gate result. Value is free text after the result
// word, such as "12/12" or "253452".
type GateCell struct {
	Gate   string `json:"gate"`
	Result string `json:"result"`
	Value  string `json:"value,omitempty"`
}

// Acceptance is one "ACCEPTED <time> | yield=<±n> <unit> | gates=G1,G2 | <summary>"
// line. Legacy lines ("ACCEPTED <time> — <summary>") have no yield or gates.
type Acceptance struct {
	Raw     string     `json:"raw"`
	At      time.Time  `json:"at"`
	Yield   *Quantity  `json:"yield,omitempty"`
	Gates   []string   `json:"gates,omitempty"`
	Results []GateCell `json:"results,omitempty"`
}

var (
	directivePattern = regexp.MustCompile(`^- (D\d+) (\S+) (.+)$`)
	directiveID      = regexp.MustCompile(`\bD\d+\b`)
	gateID           = regexp.MustCompile(`\bG\d+\b`)
	gateCellKey      = regexp.MustCompile(`^(?i)(?:gate\s+)?(G\d+)$`)
)

// ParseEpoch reads an epoch file. Unknown sections are ignored; a slice-log
// line that does not parse is kept in Malformed and never counted.
func ParseEpoch(r io.Reader) (*Epoch, error) {
	epoch := &Epoch{Fields: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for num := 1; scanner.Scan(); num++ {
		raw := strings.TrimRight(scanner.Text(), " \t")
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			if strings.HasPrefix(section, "exit gate") {
				epoch.HasExitGate = true
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		switch {
		case section == "":
			if m := fieldPattern.FindStringSubmatch(trimmed); m != nil {
				epoch.Fields[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}
		case section == "directives":
			if m := directivePattern.FindStringSubmatch(trimmed); m != nil {
				at, _ := parseWhen(m[2])
				epoch.Directives = append(epoch.Directives, Directive{ID: m[1], At: at, Text: m[3]})
			}
		case strings.HasPrefix(section, "slice log"):
			if strings.HasPrefix(trimmed, "ACCEPTED ") {
				epoch.Acceptances = append(epoch.Acceptances, parseAcceptance(trimmed))
				continue
			}
			entry, err := ParseSliceLine(trimmed)
			if err != nil {
				epoch.Malformed = append(epoch.Malformed, trimmed)
				continue
			}
			epoch.Log = append(epoch.Log, entry)
		case strings.HasPrefix(section, "exit gate"), strings.HasPrefix(section, "gate amendment"):
			epoch.parseGateLine(trimmed, num, strings.HasPrefix(section, "exit gate"))
		}
	}
	return epoch, scanner.Err()
}

// ParseSliceLine reads "<ISO time> | <what changed> | key=value | ...". The
// time, the change and an exit metric are required. Line counts default to
// zero, ack is free text whose directive IDs (D<n>) are kept, "gate G<n>"
// cells record gate results, other key=value fields are ignored, and extra
// free text joins the change.
func ParseSliceLine(line string) (SliceLine, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		return SliceLine{}, fmt.Errorf("want <time> | <what changed> | exit metric=<value>")
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
	if err != nil {
		return SliceLine{}, err
	}
	entry := SliceLine{At: at, What: strings.TrimSpace(parts[1])}
	hasMetric := false
	for _, part := range parts[2:] {
		part = strings.TrimSpace(part)
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			if part != "" {
				entry.What += " | " + part
			}
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if cell, ok := parseGateCell(key, value); ok {
			entry.Gates = append(entry.Gates, cell)
			continue
		}
		switch strings.ToLower(key) {
		case "exit metric", "metric":
			entry.Metric, hasMetric = value, true
		case "net runtime lines", "runtime lines":
			entry.RuntimeLines = leadingNumber(value)
		case "net test lines", "test lines":
			entry.TestLines = leadingNumber(value)
		case "ack", "acks":
			entry.Acks = directiveID.FindAllString(value, -1)
		}
	}
	if !hasMetric {
		return SliceLine{}, fmt.Errorf("want exit metric=<value>")
	}
	return entry, nil
}

// parseGateCell reads "gate G3" (or "G3") = "pass 253452".
func parseGateCell(key, value string) (GateCell, bool) {
	m := gateCellKey.FindStringSubmatch(key)
	if m == nil {
		return GateCell{}, false
	}
	result, rest, _ := strings.Cut(strings.TrimSpace(value), " ")
	return GateCell{Gate: strings.ToUpper(m[1]), Result: strings.ToLower(result), Value: strings.TrimSpace(rest)}, true
}

func parseAcceptance(raw string) Acceptance {
	accepted := Acceptance{Raw: raw}
	cells := strings.Split(strings.TrimSpace(strings.TrimPrefix(raw, "ACCEPTED")), "|")
	if first := strings.Fields(cells[0]); len(first) > 0 {
		accepted.At, _ = parseWhen(first[0])
	}
	for _, cell := range cells[1:] {
		key, value, ok := strings.Cut(strings.TrimSpace(cell), "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if gate, ok := parseGateCell(key, value); ok {
			accepted.Results = append(accepted.Results, gate)
			continue
		}
		switch strings.ToLower(key) {
		case "yield":
			if quantity, ok := ParseQuantity(value); ok {
				accepted.Yield = &quantity
			}
		case "gates":
			accepted.Gates = gateID.FindAllString(strings.ToUpper(value), -1)
		}
	}
	return accepted
}

// LastAcceptance returns the final ACCEPTED line, or nil.
func (e *Epoch) LastAcceptance() *Acceptance {
	if len(e.Acceptances) == 0 {
		return nil
	}
	return &e.Acceptances[len(e.Acceptances)-1]
}

// ReopenedBy lists the directives dated after the last ACCEPTED line. Only an
// orchestrator directive written after acceptance reopens an epoch; an open
// acknowledgement or a malformed log line never does.
func (e *Epoch) ReopenedBy() []string {
	accepted := e.LastAcceptance()
	if accepted == nil || accepted.At.IsZero() {
		return nil
	}
	var ids []string
	for _, directive := range e.Directives {
		if directive.At.After(accepted.At) {
			ids = append(ids, directive.ID)
		}
	}
	return ids
}

// IsAccepted is true when the epoch has an ACCEPTED line that no later
// directive reopened.
func (e *Epoch) IsAccepted() bool {
	return e.LastAcceptance() != nil && len(e.ReopenedBy()) == 0
}

// YieldEstimate reads the "- Yield estimate: <±n> <unit>" header, or nil.
func (e *Epoch) YieldEstimate() *Quantity {
	quantity, ok := ParseQuantity(e.Fields["yield estimate"])
	if !ok {
		return nil
	}
	return &quantity
}

// Yield is the epoch's measured result: the yield= cell of the last ACCEPTED
// line, else an inventory gate measured against its "from" baseline. It is
// never the sum of slice-log deltas, which workers sometimes log cumulatively.
func (e *Epoch) Yield() (*Quantity, string) {
	if accepted := e.LastAcceptance(); accepted != nil && accepted.Yield != nil {
		return accepted.Yield, "accepted"
	}
	estimate := e.YieldEstimate()
	var fallback *GateResult
	for _, gate := range EvaluateGates(e).Gates {
		if gate.Yield == nil {
			continue
		}
		if estimate != nil && normalizeUnit(estimate.Unit) != "" && sameUnit(estimate.Unit, gate.Yield.Unit) {
			return gate.Yield, "inventory gate " + gate.ID
		}
		if fallback == nil {
			fallback = &gate
		}
	}
	if fallback != nil {
		return fallback.Yield, "inventory gate " + fallback.ID
	}
	return nil, ""
}
