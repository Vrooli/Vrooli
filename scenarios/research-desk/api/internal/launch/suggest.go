package launch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"research-desk/internal/home"
)

// SuggestedQuote is one stance quote offered for a launch, with its source.
type SuggestedQuote struct {
	Text, Source, CommissionID string
}

// Suggestions are a launch's defaults for the selected topics.
type Suggestions struct {
	// Quotes are the operator's own words or nothing: the stance_quotes of
	// the newest desk-built PACKET.json naming any of the topics, else the
	// quoted lines of the STANCE-MESSAGE.txt of the newest commission on them
	// that actually started. A hand-made folder that never started is never a
	// source: its stance may be setup talk, not direction. QuotesReason says
	// where they came from or why there are none (only then is typing
	// required). A quote is offered exactly as written, ellipses included.
	Quotes       []SuggestedQuote
	QuotesReason string
	Topics       []TopicWorkstreams
}

// TopicWorkstreams is one selected topic's workstreams from mission_check.py.
type TopicWorkstreams struct {
	Topic       string
	Workstreams []home.Workstream
	Evidence    home.Evidence
}

// Suggest returns the stance quotes and workstreams for the topics.
func (s *Service) Suggest(ctx context.Context, topics []string) (Suggestions, error) {
	var out Suggestions
	if len(topics) == 0 {
		return out, nil
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{MissionCheck: true})
	if err != nil {
		return out, err
	}
	for _, name := range topics {
		tw := TopicWorkstreams{Topic: name}
		if t := snap.Topic(name); t == nil {
			tw.Evidence = home.Unknown("no topic " + name)
		} else {
			tw.Workstreams, tw.Evidence = t.Check.Workstreams, t.Check.Evidence
		}
		out.Topics = append(out.Topics, tw)
	}
	var built, started newest
	for _, c := range snap.Commissions {
		named := false
		for _, t := range topics {
			named = named || names(c, t)
		}
		if !named {
			continue
		}
		if c.Packet != nil {
			if at, err := time.Parse(time.RFC3339, c.Packet.BuiltAt); err == nil {
				built.offer(c, at)
			}
		}
		if c.Started.Known {
			started.offer(c, c.Started.Value)
		}
	}
	var tried []string
	if c := built.c; c != nil {
		if out.Quotes = packetQuotes(c); len(out.Quotes) > 0 {
			out.QuotesReason = "from " + c.ID + "'s " + home.PacketFile + " (the newest packet the desk built on these topics)"
			return out, nil
		}
		tried = append(tried, "the newest packet the desk built on these topics, "+c.ID+", has no stance_quotes")
	}
	if c := started.c; c != nil {
		var why string
		if out.Quotes, why = stanceQuotes(c); len(out.Quotes) > 0 {
			out.QuotesReason = "from the quoted lines of " + c.ID + "'s " + home.StanceFile + " (the newest commission on these topics that started)"
			return out, nil
		}
		tried = append(tried, why)
	} else {
		tried = append(tried, "no commission on these topics has started")
	}
	out.QuotesReason = strings.Join(tried, "; ") + "; type the stance quotes"
	return out, nil
}

// newest keeps the newest commission offered, ties broken by ID.
type newest struct {
	c  *home.Commission
	at time.Time
}

func (n *newest) offer(c *home.Commission, at time.Time) {
	if n.c == nil || at.After(n.at) || (at.Equal(n.at) && c.ID > n.c.ID) {
		n.c, n.at = c, at
	}
}

// packetQuotes reads the stance_quotes of a desk-built PACKET.json.
func packetQuotes(c *home.Commission) []SuggestedQuote {
	var spec struct {
		StanceQuotes []string `json:"stance_quotes"`
	}
	if json.Unmarshal(c.Packet.Spec, &spec) != nil {
		return nil
	}
	path := filepath.Join(c.Dir, home.PacketFile)
	var out []SuggestedQuote
	for _, q := range spec.StanceQuotes {
		if q = unquote(q); q != "" {
			out = append(out, SuggestedQuote{Text: q, Source: path, CommissionID: c.ID})
		}
	}
	return out
}

// stanceQuotes reads the whole quoted lines of a commission's
// STANCE-MESSAGE.txt, or says why there are none.
func stanceQuotes(c *home.Commission) ([]SuggestedQuote, string) {
	path := filepath.Join(c.Dir, home.StanceFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "the newest commission on these topics that started, " + c.ID + ", has no readable " + home.StanceFile + " (" + err.Error() + ")"
	}
	var out []SuggestedQuote
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if quoted(line) {
			if q := unquote(line); q != "" {
				out = append(out, SuggestedQuote{Text: q, Source: path, CommissionID: c.ID})
			}
		}
	}
	if len(out) == 0 {
		return nil, "the newest commission on these topics that started, " + c.ID + ", has no quoted lines in " + home.StanceFile
	}
	return out, ""
}

func quoted(line string) bool {
	r := []rune(line)
	if len(r) < 2 {
		return false
	}
	return strings.ContainsRune("\"“", r[0]) && strings.ContainsRune("\"”", r[len(r)-1])
}

func unquote(q string) string {
	q = strings.TrimSpace(q)
	if quoted(q) {
		r := []rune(q)
		q = string(r[1 : len(r)-1])
	}
	return strings.TrimSpace(q)
}
