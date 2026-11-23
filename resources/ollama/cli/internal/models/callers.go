package models

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// CallerReport is `models callers --json`: who sends requests for each model,
// as the activity edge recorded it in the shared model-usage file.
type CallerReport struct {
	Resource      string        `json:"resource"`
	TrackingSince time.Time     `json:"tracking_since"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Models        []ModelCaller `json:"models"`
}

// ModelCaller is one model's requests split by caller. Unattributed counts
// requests recorded before the edge identified callers.
type ModelCaller struct {
	Model         string        `json:"model"`
	Roles         []string      `json:"roles,omitempty"`
	Requests      int64         `json:"requests"`
	LastRequestAt time.Time     `json:"last_request_at"`
	Unattributed  int64         `json:"unattributed_requests"`
	Callers       []CallerCount `json:"callers"`
}

// CallerCount is one caller's share of a model's requests.
type CallerCount struct {
	Caller        string    `json:"caller"`
	Requests      int64     `json:"requests"`
	LastRequestAt time.Time `json:"last_request_at"`
}

// Callers prints per-model caller attribution.
func (h *Handlers) Callers(args []string) error {
	fs := flag.NewFlagSet("models callers", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Only this model (name or name:tag)")
	asJSON := fs.Bool("json", false, "Emit the caller report as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	deps := h.retentionDeps()
	paths, err := deps.Paths()
	if err != nil {
		return err
	}
	usage, err := modelretention.ReadUsage(paths.UsageFile)
	if err != nil {
		return err
	}
	roles := map[string][]string{}
	if p, err := deps.LoadPolicy(); err == nil {
		for role, binding := range p.Roles {
			key := modelretention.CanonicalName(binding.Model)
			roles[key] = append(roles[key], role)
		}
	}
	report := buildCallerReport(usage, roles, *model)
	if *asJSON {
		return writeJSON(h.Stdout, report)
	}
	if report.TrackingSince.IsZero() {
		fmt.Fprintln(h.Stdout, "No usage recorded yet: the Ollama activity edge has not written model usage.")
		return nil
	}
	fmt.Fprintf(h.Stdout, "Requests through the activity edge since %s\n", report.TrackingSince.Local().Format(time.RFC3339))
	for _, m := range report.Models {
		roleText := ""
		if len(m.Roles) > 0 {
			roleText = "  roles: " + strings.Join(m.Roles, ", ")
		}
		fmt.Fprintf(h.Stdout, "\n%s  requests=%d  last=%s%s\n", m.Model, m.Requests, m.LastRequestAt.Local().Format(time.RFC3339), roleText)
		for _, c := range m.Callers {
			fmt.Fprintf(h.Stdout, "  %8d  %s  %s\n", c.Requests, c.LastRequestAt.Local().Format(time.RFC3339), c.Caller)
		}
		if m.Unattributed > 0 {
			fmt.Fprintf(h.Stdout, "  %8d  (before caller tracking)\n", m.Unattributed)
		}
	}
	return nil
}

func buildCallerReport(usage modelretention.UsageDocument, roles map[string][]string, only string) CallerReport {
	report := CallerReport{Resource: usage.Resource, TrackingSince: usage.TrackingSince, UpdatedAt: usage.UpdatedAt, Models: []ModelCaller{}}
	want := modelretention.CanonicalName(only)
	for name, entry := range usage.Models {
		if want != "" && name != want {
			continue
		}
		m := ModelCaller{Model: name, Requests: entry.Requests, LastRequestAt: entry.LastRequestAt, Callers: []CallerCount{}}
		m.Roles = append(m.Roles, roles[name]...)
		sort.Strings(m.Roles)
		var attributed int64
		for caller, usage := range entry.Callers {
			m.Callers = append(m.Callers, CallerCount{Caller: caller, Requests: usage.Requests, LastRequestAt: usage.LastRequestAt})
			attributed += usage.Requests
		}
		sort.Slice(m.Callers, func(i, j int) bool {
			if m.Callers[i].Requests != m.Callers[j].Requests {
				return m.Callers[i].Requests > m.Callers[j].Requests
			}
			return m.Callers[i].Caller < m.Callers[j].Caller
		})
		if attributed < m.Requests {
			m.Unattributed = m.Requests - attributed
		}
		report.Models = append(report.Models, m)
	}
	sort.Slice(report.Models, func(i, j int) bool {
		if report.Models[i].Requests != report.Models[j].Requests {
			return report.Models[i].Requests > report.Models[j].Requests
		}
		return report.Models[i].Model < report.Models[j].Model
	})
	return report
}
