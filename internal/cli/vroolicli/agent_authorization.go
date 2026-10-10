package vroolicli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// runAgentAuthorization never exchanges the caller's OS account for a human
// identity. It prepares reviewable consent and directs the owner to the
// configured identity provider; approval happens at the authority boundary.
func (app *App) runAgentAuthorization(ctx *AppContext, action string, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent " + action)
	fs.SetOutput(ctx.Stderr)
	duration := fs.Duration("expires-in", 24*time.Hour, "grant lifetime (at most 168h; ends early when the session or run ends)")
	approvalBase := fs.String("approval-base", strings.TrimSpace(os.Getenv("AGENT_MANAGER_AUTHORIZATION_URL")), "authenticated Agent Manager browser URL")
	rules := &authorizationRules{}
	fs.Var(authorizationPolicyFlag{rules}, "policy", "reviewed interactive policy (default swarm-bookkeeping); repeat to ask for several in one consent, each followed by its own --target")
	fs.Var(authorizationTargetFlag{rules}, "target", "explicit domain:object or domain:* for the preceding --policy (repeatable)")
	// Standard flag parsing stops at positional arguments; permit the documented
	// session-first spelling while still rejecting multiple selectors.
	selector := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		selector = args[0]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if selector == "" && fs.NArg() == 1 {
		selector = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent "+action, "exactly one session or run is required")
	}
	if selector == "" {
		return clipolicy.UsageErrorf("agent "+action, "session or run required; use vrooli agent list --json")
	}
	base, err := discovery.ResolveScenarioURLDefault(context.Background(), "agent-manager")
	if err != nil {
		return fmt.Errorf("resolve Agent Manager: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{}}

	home, err := configpkg.HomeDir()
	if err != nil {
		return err
	}
	leases, err := maintenance.NewController(ctx.Root, home).ListAgentSessions()
	if err != nil {
		return err
	}
	lease, err := selectAuthorizationLease(leases, selector)
	if err != nil {
		return err
	}
	if _, err := cliutil.ObserveProcessBinding(lease.PID); err != nil {
		return fmt.Errorf("session process is no longer live: %w", err)
	}
	if action == "revoke-authorization" {
		if *approvalBase == "" {
			return errors.New("set --approval-base to the authenticated Agent Manager browser URL")
		}
		data, err := authorizationRequest(client, http.MethodGet, base+"/api/v1/authorizations?pid="+strconv.Itoa(lease.PID), nil)
		if err != nil {
			return err
		}
		var grants []struct{ ID, State string }
		if err := json.Unmarshal(data, &grants); err != nil {
			return err
		}
		for _, grant := range grants {
			if grant.State == "approved" {
				approval, err := authorizationDecisionURL(*approvalBase, grant.ID, "revoke")
				if err != nil {
					return err
				}
				fmt.Fprintln(ctx.Stdout, approval)
				return nil
			}
		}
		return errors.New("this session has no approved grant to revoke")
	}
	if action == "authorization" {
		data, err := authorizationRequest(client, http.MethodGet, base+"/api/v1/authorizations?pid="+strconv.Itoa(lease.PID), nil)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(ctx.Stdout, string(data))
		return err
	}
	requested, err := rules.resolve()
	if err != nil {
		return clipolicy.UsageErrorf("agent authorize", "%s", err.Error())
	}
	if *approvalBase == "" {
		return errors.New("set --approval-base (or AGENT_MANAGER_AUTHORIZATION_URL) to the authenticated Agent Manager browser URL; tunnel reachability alone is not human proof")
	}
	if _, err := authorizationDecisionURL(*approvalBase, "request", "approve"); err != nil {
		return err
	}
	body := map[string]any{"pid": lease.PID, "harness_kind": lease.Agent, "harness_session": lease.SessionID, "duration_seconds": int64(duration.Seconds())}
	if len(requested) == 1 {
		body["policy"], body["targets"] = requested[0].Policy, requested[0].Targets
	} else {
		body["rules"] = requested
	}
	if lease.Owner.Kind == "agent-manager-run" {
		body["run_id"] = lease.Owner.Ref
	}
	data, err := authorizationRequest(client, http.MethodPost, base+"/api/v1/authorizations/requests", body)
	if err != nil {
		return err
	}
	var grant struct {
		ID               string    `json:"id"`
		RequestExpiresAt time.Time `json:"request_expires_at"`
	}
	if err := json.Unmarshal(data, &grant); err != nil || grant.ID == "" {
		return errors.New("invalid authorization receipt")
	}
	approval, err := authorizationDecisionURL(*approvalBase, grant.ID, "approve")
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Authorization request %s is pending human consent.\n%s\nApprove by: %s\nPermission lifetime starts at approval. After approval, this running session uses the grant on subsequent CLI requests.\nA newer request replaces this one while it is pending, and a newer approval replaces this session's earlier grant: ask for every policy the work needs in one request (repeat --policy, each with its --target).\n", grant.ID, approval, grant.RequestExpiresAt.Format(time.RFC3339))
	return nil
}

func selectAuthorizationLease(leases []scenarioruntime.EditorLease, selector string) (scenarioruntime.EditorLease, error) {
	var selected *scenarioruntime.EditorLease
	for i := range leases {
		lease := &leases[i]
		if lease.Status != scenarioruntime.EditorLeaseActive {
			continue
		}
		// The session ID, owning run, systemd scope (VROOLI_AGENT_SESSION) and
		// PID all name the same live, kernel-verified lease.
		if selector != lease.SessionID && selector != lease.Owner.Ref && selector != lease.Scope && selector != strconv.Itoa(lease.PID) {
			continue
		}
		if selected != nil {
			return scenarioruntime.EditorLease{}, errors.New("selector matches multiple sessions; use the exact session ID")
		}
		selected = lease
	}
	if selected == nil {
		return scenarioruntime.EditorLease{}, fmt.Errorf("no live host lease for %s; use vrooli agent list --json", selector)
	}
	return *selected, nil
}

// authorizationRule is one reviewed policy and its targets, in request order.
type authorizationRule struct {
	Policy  string   `json:"policy"`
	Targets []string `json:"targets"`
}

// authorizationRules keeps --policy and --target in command-line order so
// each target belongs to the policy before it.
type authorizationRules struct {
	rules   []authorizationRule
	orphans []string
}

type (
	authorizationPolicyFlag struct{ r *authorizationRules }
	authorizationTargetFlag struct{ r *authorizationRules }
)

func (f authorizationPolicyFlag) String() string { return "" }

func (f authorizationPolicyFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("--policy cannot be empty")
	}
	f.r.rules = append(f.r.rules, authorizationRule{Policy: value})
	return nil
}

func (f authorizationTargetFlag) String() string { return "" }

func (f authorizationTargetFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("--target cannot be empty")
	}
	if n := len(f.r.rules); n > 0 {
		f.r.rules[n-1].Targets = append(f.r.rules[n-1].Targets, value)
	} else {
		f.r.orphans = append(f.r.orphans, value)
	}
	return nil
}

// resolve applies the single-policy spellings: targets before the only
// --policy belong to it, and no --policy means swarm-bookkeeping.
func (r *authorizationRules) resolve() ([]authorizationRule, error) {
	rules := append([]authorizationRule(nil), r.rules...)
	switch {
	case len(rules) == 0:
		rules = []authorizationRule{{Policy: "swarm-bookkeeping"}}
		fallthrough
	case len(rules) == 1:
		rules[0].Targets = append(append([]string(nil), r.orphans...), rules[0].Targets...)
	case len(r.orphans) > 0:
		return nil, errors.New("with several --policy flags, put each --target after the --policy it belongs to")
	}
	for _, rule := range rules {
		if len(rule.Targets) == 0 {
			return nil, fmt.Errorf("--policy %s needs at least one --target (for example --target swarm-manager:*)", rule.Policy)
		}
	}
	return rules, nil
}

func authorizationDecisionURL(base, id, action string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errors.New("invalid authenticated approval URL")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/authorizations/" + url.PathEscape(id) + "/" + action
	return u.String(), nil
}

func authorizationRequest(client *http.Client, method, url string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(response.StatusCode, data)
	}
	return data, nil
}
