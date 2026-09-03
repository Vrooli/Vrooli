package vroolicli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
)

// `vrooli agent thaw <scope>` reverses contain-storm: it thaws the agent
// session scope through platform-go (which refuses anything outside
// vrooli-agents.slice), records a `thawed` decision beside the freeze in the
// runtime registry, and resolves the storm incidents that name the scope
// through the autoheal API when it answers. The thaw itself never depends on
// the API: a frozen terminal must be recoverable with the API down.
const (
	thawDecisionScenario = "vrooli-agents"
	thawEpochID          = "no-epoch"
	thawAPITimeout       = 10 * time.Second
	thawProbeTimeout     = 5 * time.Second
	autohealScenario     = "vrooli-autoheal"
	httpRedirectFloor    = 300
)

func (app *App) runAgentThaw(ctx *CommandContext, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent thaw")
	fs.SetOutput(ctx.Stderr)
	note := fs.String("note", "operator thawed the scope with vrooli agent thaw", "note recorded on the resolved incident")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		return clipolicy.UsageErrorf("agent thaw", "exactly one scope is required: a vrooli-agent-* unit name or a cgroup:<path> ref")
	}
	runCtx, cancel := context.WithTimeout(context.Background(), thawAPITimeout)
	defer cancel()
	ref, err := agentScopeRef(runCtx, fs.Arg(0), shell.OSRunner{})
	if err != nil {
		return err
	}
	if err := platformgo.ThawScope(ref); err != nil {
		return err
	}
	name := scopeUnitName(ref)
	if ctx.Stdout != nil {
		_, _ = fmt.Fprintf(ctx.Stdout, "thawed %s (%s)\n", name, ref.String())
	}
	if err := recordThawDecision(runCtx, ref, *note); err != nil && ctx.Stderr != nil {
		_, _ = fmt.Fprintf(ctx.Stderr, "thaw recorded no decision row: %v\n", err)
	}
	resolved, err := resolveStormIncidents(runCtx, name, *note)
	if ctx.Stdout != nil {
		switch {
		case err != nil:
			_, _ = fmt.Fprintf(ctx.Stdout, "incidents not resolved through the autoheal API (%v); the scope is thawed regardless\n", err)
		case len(resolved) == 0:
			_, _ = fmt.Fprintln(ctx.Stdout, "no open storm incident names this scope")
		default:
			_, _ = fmt.Fprintf(ctx.Stdout, "resolved incident(s) through the autoheal API: %s\n", strings.Join(resolved, ", "))
		}
	}
	return nil
}

// agentScopeRef resolves the operator's argument: a "cgroup:<path>" ref is
// used as written; a unit name is resolved through the user manager's
// ControlGroup so the thaw never guesses a cgroup path.
func agentScopeRef(ctx context.Context, argument string, runner shell.Runner) (platformgo.ScopeRef, error) {
	argument = strings.TrimSpace(argument)
	if strings.Contains(argument, ":") {
		return platformgo.ParseScopeRef(argument)
	}
	if argument == "" {
		return platformgo.ScopeRef{}, fmt.Errorf("agent thaw: a scope is required")
	}
	unit := argument
	if !strings.HasSuffix(unit, ".scope") {
		unit += ".scope"
	}
	if !strings.HasPrefix(unit, "vrooli-agent-") {
		return platformgo.ScopeRef{}, fmt.Errorf("agent thaw: %q is not an agent session scope (vrooli-agent-*)", argument)
	}
	probeCtx, cancel := context.WithTimeout(ctx, thawProbeTimeout)
	defer cancel()
	out, err := runner.Run(probeCtx, "systemctl", "--user", "show", "-p", "ControlGroup", "--value", unit)
	if err != nil {
		return platformgo.ScopeRef{}, fmt.Errorf("agent thaw: resolve %s: %w: %s", unit, err, strings.TrimSpace(string(out)))
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return platformgo.ScopeRef{}, fmt.Errorf("agent thaw: %s has no control group (is it running?)", unit)
	}
	return platformgo.ScopeRef{Name: strings.TrimSuffix(unit, ".scope"), Kind: platformgo.ScopeKindCgroup, Path: path}, nil
}

func scopeUnitName(ref platformgo.ScopeRef) string {
	if ref.Name != "" {
		return ref.Name + ".scope"
	}
	if idx := strings.LastIndex(ref.Path, "/"); idx >= 0 {
		return ref.Path[idx+1:]
	}
	return ref.String()
}

func recordThawDecision(ctx context.Context, ref platformgo.ScopeRef, note string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return err
	}
	defer store.Close()
	epoch := thawEpochID
	if epochs, err := store.ListPressureEpochs(ctx, 1); err == nil && len(epochs) > 0 {
		epoch = epochs[0].EpochID
	}
	details, _ := json.Marshal(map[string]any{"scope_name": scopeUnitName(ref), "scope_path": ref.Path, "note": note})
	_, err = store.RecordRecoveryDecision(ctx, scenarioruntime.RecoveryDecision{
		EpochID:        epoch,
		Scenario:       thawDecisionScenario,
		State:          "thawed",
		Reason:         fmt.Sprintf("thawed %s: %s", scopeUnitName(ref), note),
		IdempotencyKey: fmt.Sprintf("%s/%s/thaw/%d", epoch, scopeUnitName(ref), time.Now().Unix()),
		DetailsJSON:    string(details),
	})
	return err
}

// resolveStormIncidents resolves every open autoheal incident whose title
// names the scope. Only the API owns incident state; nothing here touches
// autoheal's database.
func resolveStormIncidents(ctx context.Context, scopeUnit, note string) ([]string, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, autohealScenario)
	if err != nil {
		return nil, err
	}
	base = strings.TrimRight(base, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/incidents?status=open&type=host_pressure&limit=200", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: thawAPITimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list incidents: %s", response.Status)
	}
	var listed struct {
		Incidents []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"incidents"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		return nil, err
	}
	short := strings.TrimSuffix(scopeUnit, ".scope")
	var resolved []string
	for _, incident := range listed.Incidents {
		if !strings.Contains(incident.Title, scopeUnit) && !strings.Contains(incident.Title, short) {
			continue
		}
		body, _ := json.Marshal(map[string]string{"note": note})
		resolve, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/incidents/"+incident.ID+"/resolve", bytes.NewReader(body))
		if err != nil {
			return resolved, err
		}
		resolve.Header.Set("Content-Type", "application/json")
		reply, err := client.Do(resolve)
		if err != nil {
			return resolved, err
		}
		_ = reply.Body.Close()
		if reply.StatusCode >= httpRedirectFloor {
			return resolved, fmt.Errorf("resolve %s: %s", incident.ID, reply.Status)
		}
		resolved = append(resolved, incident.ID)
	}
	return resolved, nil
}
