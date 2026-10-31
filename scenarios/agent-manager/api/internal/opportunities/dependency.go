package opportunities

import (
	"context"
	"math"
	"strings"
	"unicode"
)

const (
	// MaxDependencySamples bounds the occurrences of one motif whose output
	// text is read to measure result dependency: enough for a stable rate,
	// few enough that a report reads tens of events, not a week of them.
	MaxDependencySamples = 30
	// MinDependency is the share of a motif's later steps that must have taken
	// an argument from an earlier step's output for a program to be able to
	// pick them. Below it the agent chose each step itself, and a program
	// would only run the calls the agent still decides. A verdict needs the
	// whole 95% Wilson interval of the sampled rate on one side of it.
	MinDependency = 0.5
	minArgChars   = 3
	wilsonZ       = 1.96
)

// Outputs reads the text the agent saw for result event ids; an id it cannot
// read is absent, and an error says why none could be read. Nil means no
// output text is available.
type Outputs func(eventIDs []string) (map[string]string, error)

// dependency is how many of a motif's later steps with an argument of their
// own could be checked against earlier output, and how many of those took an
// argument from it. Open counts the later steps with such arguments whose
// earlier output could not be read, and Unread says why when reading failed.
type dependency struct {
	Measured, Used, Open int
	Unread               string
}

func (d dependency) rate() float64 {
	if d.Measured == 0 {
		return 0
	}
	return float64(d.Used) / float64(d.Measured)
}

// interval is the Wilson 95% interval of the dependency rate over the measured
// steps: a small sample leaves it wide, so a rate near the threshold decides
// nothing.
func (d dependency) interval() (low, high float64) {
	if d.Measured == 0 {
		return 0, 1
	}
	n, p, z2 := float64(d.Measured), d.rate(), wilsonZ*wilsonZ
	centre := (p + z2/(2*n)) / (1 + z2/n)
	half := wilsonZ * math.Sqrt(p*(1-p)/n+z2/(4*n*n)) / (1 + z2/n)
	return max(0, centre-half), min(1, centre+half)
}

// varying is whether any later step has an argument that differs between
// occurrences: only those arguments are chosen per occurrence, and so only
// they can come from an earlier output. A step whose arguments never change
// is fixed in the program.
func (d dependency) varying() bool { return d.Measured+d.Open > 0 }

// argTokens are the identifying words of a step's arguments: what follows the
// command path, minus flags (a flag's `=value` kept) and short words.
func argTokens(s step) []string {
	rest := s.command
	if at := strings.Index(rest, s.path); at >= 0 {
		rest = rest[at+len(s.path):]
	}
	var out []string
	for _, word := range strings.Fields(rest) {
		if strings.HasPrefix(word, "-") {
			eq := strings.IndexByte(word, '=')
			if eq < 0 {
				continue
			}
			word = word[eq+1:]
		}
		word = strings.Trim(word, `"'`+"`,;()[]{}")
		if len(word) >= minArgChars {
			out = append(out, word)
		}
	}
	return out
}

// hitText is what an output returns as results: without the bodies of its
// documents (CDATA sections; one left open by a cut output runs to the end)
// and without backticked spans, where outputs print the commands and
// identifiers they merely suggest ("read `x` next"). Spans pair backticks
// within a line, so a stray backtick hides nothing after it. Other prose
// around a result still counts as one.
func hitText(text string) string {
	var kept strings.Builder
	for {
		start := strings.Index(text, "<![CDATA[")
		if start < 0 {
			kept.WriteString(text)
			break
		}
		kept.WriteString(text[:start])
		end := strings.Index(text[start:], "]]>")
		if end < 0 {
			break
		}
		text = text[start+end+len("]]>"):]
	}
	lines := strings.Split(kept.String(), "\n")
	for li, line := range lines {
		parts := strings.Split(line, "`")
		for i := 1; i < len(parts); i += 2 {
			if i == len(parts)-1 {
				break // an unmatched backtick opens no span
			}
			parts[i] = ""
		}
		lines[li] = strings.Join(parts, " ")
	}
	return strings.Join(lines, "\n")
}

// mentions is whether text holds token as a whole identifier, not inside a
// longer one.
func mentions(text, token string) bool {
	for from := 0; ; {
		at := strings.Index(text[from:], token)
		if at < 0 {
			return false
		}
		start, end := from+at, from+at+len(token)
		if (start == 0 || !identRune(rune(text[start-1]))) && (end == len(text) || !identRune(rune(text[end]))) {
			return true
		}
		from = start + 1
	}
}

func identRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' }

// measureDependency samples at most MaxDependencySamples occurrences, evenly
// spaced, and for each later step's per-occurrence arguments checks whether
// an earlier step of the same occurrence printed them.
func (m motif) measureDependency(outputs Outputs) dependency {
	var d dependency
	places := m.places
	if len(places) > MaxDependencySamples {
		sampled := make([][]step, MaxDependencySamples)
		for i := range sampled {
			sampled[i] = places[i*len(places)/MaxDependencySamples]
		}
		places = sampled
	}
	chosen := make([][][]string, len(places))
	var ids []string
	for pi, place := range places {
		chosen[pi] = make([][]string, len(place))
		for k := 1; k < len(place); k++ {
			chosen[pi][k] = withoutTyped(m.chosenTokens(k, place[k]), place[:k])
			if len(chosen[pi][k]) > 0 {
				for _, earlier := range place[:k] {
					ids = append(ids, earlier.output)
				}
			}
		}
	}
	var texts map[string]string
	if outputs != nil && len(ids) > 0 {
		var err error
		if texts, err = outputs(ids); err != nil {
			d.Unread = err.Error()
		}
	}
	hits := make(map[string]string, len(texts))
	for id, text := range texts {
		hits[id] = hitText(text)
	}
	for pi, place := range places {
		for k := 1; k < len(place); k++ {
			tokens := chosen[pi][k]
			if len(tokens) == 0 {
				continue
			}
			readable, used := false, false
			for _, earlier := range place[:k] {
				text, ok := hits[earlier.output]
				if earlier.output == "" || !ok {
					continue
				}
				readable = true
				for _, token := range tokens {
					used = used || mentions(text, token)
				}
			}
			switch {
			case !readable:
				d.Open++
			case used:
				d.Measured++
				d.Used++
			default:
				d.Measured++
			}
		}
	}
	return d
}

// withoutTyped drops the words an earlier step of the occurrence already had
// in its own command line: the agent typed them, so a later use of them is not
// derived from any output that echoes them back.
func withoutTyped(tokens []string, earlier []step) []string {
	var out []string
	for _, token := range tokens {
		typed := false
		for _, e := range earlier {
			for _, own := range argTokens(e) {
				typed = typed || own == token
			}
		}
		if !typed {
			out = append(out, token)
		}
	}
	return out
}

// chosenTokens are step k's argument words that are not the same in every
// occurrence of the motif.
func (m motif) chosenTokens(k int, s step) []string {
	var out []string
	for _, token := range argTokens(s) {
		fixed := true
		for _, place := range m.places {
			same := false
			for _, other := range argTokens(place[k]) {
				same = same || other == token
			}
			fixed = fixed && same
		}
		if !fixed {
			out = append(out, token)
		}
	}
	return out
}

// OutputStore reads result text by event id.
type OutputStore interface {
	ToolOutputs(ctx context.Context, eventIDs []string) (map[string]string, error)
}

// OutputsFrom binds a store that can read result text to ctx; a read error is
// returned, so the sequences it leaves unmeasured say why; any other source
// leaves dependency unmeasurable (nil). A read that fails reads as no text.
func OutputsFrom(ctx context.Context, source any) Outputs {
	store, ok := source.(OutputStore)
	if !ok {
		return nil
	}
	return func(ids []string) (map[string]string, error) {
		return store.ToolOutputs(ctx, ids)
	}
}
