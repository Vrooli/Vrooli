package teams

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/attribution"
)

// Reported identity is observation metadata, never runtime writer authority.
type frictionReport struct {
	Scope            string             `json:"scope"`
	Severity         string             `json:"severity"`
	Slug             string             `json:"slug"`
	Reporter         string             `json:"reporter"`
	ReporterTeam     string             `json:"reporter_team"`
	ObservedAt       string             `json:"observed_at"`
	Context          map[string]*string `json:"context"`
	Expected         string             `json:"expected"`
	Actual           string             `json:"actual"`
	Description      string             `json:"description"`
	HonestyFlags     []string           `json:"honesty_flags"`
	Attempt          string             `json:"attempt"`
	Observation      string             `json:"observation"`
	Explanation      string             `json:"explanation"`
	RecurrenceCount  int                `json:"recurrence_count,omitempty"`
	PriorEntry       string             `json:"prior_entry,omitempty"`
	CurrentlyBlocked bool               `json:"currently_blocked,omitempty"`
}

var frictionSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (r frictionReport) validate() error {
	if !frictionSlug.MatchString(r.Slug) || len(r.Slug) > 120 {
		return fmt.Errorf("slug must be kebab-case, at most 120 characters")
	}
	switch r.Scope {
	case "toolchain", "run-execution", "prompt-team-agent-storage", "recurring-workaround", "unknown":
	default:
		return fmt.Errorf("invalid friction scope")
	}
	switch r.Severity {
	case "blocking", "recurring", "one-off":
	default:
		return fmt.Errorf("invalid friction severity")
	}
	for _, field := range []struct{ name, value string }{{"reporter", r.Reporter}, {"reporter_team", r.ReporterTeam}, {"expected", r.Expected}, {"actual", r.Actual}, {"description", r.Description}, {"attempt", r.Attempt}, {"observation", r.Observation}, {"explanation", r.Explanation}} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	date, err := time.Parse("2006-01-02", r.ObservedAt)
	if err != nil || date.After(time.Now().UTC()) {
		return fmt.Errorf("observed_at must be a non-future YYYY-MM-DD date")
	}
	for key := range r.Context {
		switch key {
		case "scenario", "skill", "member", "command", "doc", "task":
		default:
			return fmt.Errorf("unknown context anchor %q", key)
		}
	}
	for _, flag := range r.HonestyFlags {
		switch flag {
		case "speculative-cause", "repeats-existing-friction-topic", "minimal-context", "auto-generated":
		default:
			return fmt.Errorf("invalid honesty flag %q", flag)
		}
	}
	if r.Severity == "recurring" && r.RecurrenceCount < 2 && strings.TrimSpace(r.PriorEntry) == "" {
		return fmt.Errorf("recurring requires recurrence_count >= 2 or prior_entry")
	}
	if r.Severity == "blocking" && !r.CurrentlyBlocked {
		return fmt.Errorf("blocking requires currently_blocked=true")
	}
	return nil
}

func (r frictionReport) content() string {
	// JSON quoted strings are YAML scalars too. Preserve multiline and special
	// characters exactly rather than interpolating user values into YAML syntax.
	scalar := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	var b strings.Builder
	b.WriteString("---\n")
	for _, f := range []struct {
		name  string
		value any
	}{{"severity", r.Severity}, {"scope", r.Scope}, {"reporter", r.Reporter}, {"reporter_team", r.ReporterTeam}, {"observed_at", r.ObservedAt}} {
		fmt.Fprintf(&b, "%s: %s\n", f.name, scalar(f.value))
	}
	b.WriteString("context:\n")
	for _, key := range []string{"scenario", "skill", "member", "command", "doc", "task"} {
		fmt.Fprintf(&b, "  %s: %s\n", key, scalar(r.Context[key]))
	}
	flags := r.HonestyFlags
	if flags == nil {
		flags = []string{}
	}
	for _, f := range []struct {
		name  string
		value any
	}{{"expected", r.Expected}, {"actual", r.Actual}, {"description", r.Description}, {"honesty_flags", flags}, {"recurrence_count", r.RecurrenceCount}, {"prior_entry", r.PriorEntry}, {"currently_blocked", r.CurrentlyBlocked}} {
		fmt.Fprintf(&b, "%s: %s\n", f.name, scalar(f.value))
	}
	fmt.Fprintf(&b, "---\n\n%s\n\n%s\n\n%s\n", r.Attempt, r.Observation, r.Explanation)
	return b.String()
}

func captureFrictionFile(ctx appctx.Context, path string, jsonOut bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var report frictionReport
	if err := decoder.Decode(&report); err != nil {
		return fmt.Errorf("invalid report JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("report file must contain one JSON object")
	}
	if err := report.validate(); err != nil {
		return err
	}
	topic := "friction-inbox/" + report.Scope + "/" + report.Slug
	content := report.content()
	// Bounded serial retry, not a concurrent exactly-once guarantee. Keep the
	// stable topic and content after uncertain writes; never invent a new slug.
	var existing KnowledgeListResponse
	query := url.Values{"topic": {topic}, "last": {"500"}}
	if err := ctx.GetWithQuery("/teams/meta-optimization/knowledge", query, &existing); err != nil {
		return fmt.Errorf("cannot establish prior capture; no write attempted: %w", err)
	}
	var entry KnowledgeEntry
	for _, candidate := range existing.Entries {
		if candidate.Topic != topic {
			continue
		}
		if candidate.Content != content {
			return fmt.Errorf("stable friction topic already has different content; reconcile existing case before writing")
		}
		entry = candidate
	}
	if entry.ID == "" {
		post := func() error {
			return ctx.Post("/teams/meta-optimization/knowledge", AddKnowledgeRequest{Topic: topic, Content: content, Source: "prompt-manager friction capture", CallerNote: "filed via report-friction skill"}, &entry)
		}
		if err := attribution.WithWriterSkill("report-friction", "meta-optimization", post); err != nil {
			return fmt.Errorf("capture outcome unresolved: %w; read knowledge-list meta-optimization --topic=%s before retrying the same report", err, topic)
		}
	}
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(entry)
	}
	fmt.Printf("Filed friction %s\n", entry.ID)
	return nil
}
