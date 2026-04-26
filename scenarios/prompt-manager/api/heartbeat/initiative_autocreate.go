package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"prompt-manager/store"
)

// initiativeNameRegex enforces kebab-case (lowercase letters, digits, hyphens),
// matching the swarm-manager initiative naming convention.
var initiativeNameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// allowedTargetScenarios is the allowlist of scenarios that can host a
// generated initiative. Starts narrow; extend as new targets emerge.
var allowedTargetScenarios = map[string]bool{
	"swarm-manager": true,
}

// defaultTargetScenario is used when InitiativeMetadata.TargetScenario is
// empty.
const defaultTargetScenario = "swarm-manager"

// validateInitiativeMetadata enforces the contract defined in
// docs/reference/decision-initiative-proposal-contract.md.
//
// - Name is required and must match initiativeNameRegex.
// - Priority must be 0 (unset) or in [1, 10].
// - DependsOn entries are non-empty and trimmed.
// - TargetScenario, if set, must be in the allowlist.
func validateInitiativeMetadata(m *store.DecisionInitiativeMetadata) error {
	if m == nil {
		return nil
	}
	name := strings.TrimSpace(m.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !initiativeNameRegex.MatchString(name) {
		return fmt.Errorf("name %q must be kebab-case (lowercase letters, digits, hyphens)", name)
	}
	if m.Priority != 0 && (m.Priority < 1 || m.Priority > 10) {
		return fmt.Errorf("priority %d must be 0 (unset) or in 1-10", m.Priority)
	}
	for i, d := range m.DependsOn {
		if strings.TrimSpace(d) == "" {
			return fmt.Errorf("depends_on[%d] must be a non-empty string", i)
		}
	}
	target := strings.TrimSpace(m.TargetScenario)
	if target != "" && !allowedTargetScenarios[target] {
		return fmt.Errorf("target_scenario %q is not in the allowlist", target)
	}
	return nil
}

// initiativeCreateRequest matches swarm-manager's initiatives.CreateRequest
// JSON shape.
type initiativeCreateRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

// initiativeCreateResponse minimally captures the field we need from
// swarm-manager's create response.
type initiativeCreateResponse struct {
	Name string `json:"name"`
}

// AutoCreateOutcome carries the result of the auto-create attempt back to
// the operator surface (CLI/UI). On success, InitiativeRef is populated.
// On failure, Error and the WorkaroundCommand / ResolveCommand strings are
// populated so the operator can run them verbatim.
type AutoCreateOutcome struct {
	Status             string `json:"status"`                         // "created" | "failed"
	InitiativeRef      string `json:"initiative_ref,omitempty"`       // "<scenario>/<name>" on success
	Error              string `json:"error,omitempty"`                // failure reason
	WorkaroundCommand  string `json:"workaround_command,omitempty"`   // pre-filled `swarm-manager initiatives create ...`
	ResolveCommand     string `json:"resolve_command,omitempty"`      // pre-filled `prompt-manager team decision-update ...`
	DescriptionTmpFile string `json:"description_tmp_file,omitempty"` // path to materialised description body
	TargetScenario     string `json:"target_scenario,omitempty"`
	InitiativeName     string `json:"initiative_name,omitempty"`
	Priority           int    `json:"priority,omitempty"`
}

// buildInitiativeDescription assembles the initiative's description body
// from the decision's topic, the selected option's label/rationale, and
// the modifications block (if any).
func buildInitiativeDescription(d *store.DecisionEntry) string {
	var b strings.Builder
	if strings.TrimSpace(d.Topic) != "" {
		b.WriteString(d.Topic)
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(d.Description) != "" {
		b.WriteString(d.Description)
		b.WriteString("\n\n")
	}
	// Selected option's label + rationale.
	if d.Selected != "" {
		for _, opt := range d.Options {
			if opt.Key == d.Selected {
				if strings.TrimSpace(opt.Label) != "" {
					fmt.Fprintf(&b, "Selected option %s: %s\n", opt.Key, opt.Label)
				}
				if strings.TrimSpace(opt.Rationale) != "" {
					b.WriteString(opt.Rationale)
					b.WriteString("\n\n")
				}
				break
			}
		}
	}
	// Modifications block.
	if d.Modifications != nil {
		m := d.Modifications
		b.WriteString("Modifications:\n")
		if len(m.ExcludedClauses) > 0 {
			b.WriteString("  Excluded clauses:\n")
			for _, c := range m.ExcludedClauses {
				fmt.Fprintf(&b, "    - %s\n", c)
			}
		}
		if len(m.Additions) > 0 {
			b.WriteString("  Additions:\n")
			for _, c := range m.Additions {
				fmt.Fprintf(&b, "    + %s\n", c)
			}
		}
		if strings.TrimSpace(m.Rationale) != "" {
			fmt.Fprintf(&b, "  Rationale: %s\n", m.Rationale)
		}
	}
	return strings.TrimSpace(b.String())
}

// resolvedTargetScenario returns the explicit target_scenario or the default.
func resolvedTargetScenario(m *store.DecisionInitiativeMetadata) string {
	if m == nil || strings.TrimSpace(m.TargetScenario) == "" {
		return defaultTargetScenario
	}
	return strings.TrimSpace(m.TargetScenario)
}

// resolvedInitiativeTitle returns the explicit title override or falls back
// to the decision's topic, then to the initiative name.
func resolvedInitiativeTitle(d *store.DecisionEntry, m *store.DecisionInitiativeMetadata) string {
	if m != nil && strings.TrimSpace(m.Title) != "" {
		return strings.TrimSpace(m.Title)
	}
	if strings.TrimSpace(d.Topic) != "" {
		return strings.TrimSpace(d.Topic)
	}
	return m.Name
}

// shellEscape minimally quotes a string for inclusion in a copy-pasteable
// shell command. Wraps in single quotes and escapes any embedded single
// quotes via the standard `'\''` trick. Empty strings render as ''.
func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`!*?[]{}()&|<>;#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildWorkaroundCommand renders the exact `swarm-manager initiatives create`
// command line the operator should run after a failed auto-create. Per d8=C,
// no retry command exists; this is the manual recovery path.
func buildWorkaroundCommand(req initiativeCreateRequest, descriptionFile, target string) string {
	var b strings.Builder
	b.WriteString(target)
	b.WriteString(" initiatives create --data ")
	payload := map[string]any{
		"name":  req.Name,
		"title": req.Title,
	}
	if req.Priority != 0 {
		payload["priority"] = req.Priority
	}
	if len(req.DependsOn) > 0 {
		payload["depends_on"] = req.DependsOn
	}
	if descriptionFile != "" {
		payload["description"] = "@" + descriptionFile
	} else if req.Description != "" {
		payload["description"] = req.Description
	}
	data, _ := json.Marshal(payload)
	b.WriteString(shellEscape(string(data)))
	return b.String()
}

// buildResolveCommand renders the `prompt-manager team decision-update`
// invocation that flips a failed auto-create to created after the operator
// has run the workaround command.
func buildResolveCommand(teamID, decisionID, initiativeRef string) string {
	return fmt.Sprintf(
		"prompt-manager team decision-update %s %s --auto-create-status=created --auto-create-initiative-ref=%s",
		shellEscape(teamID), shellEscape(decisionID), shellEscape(initiativeRef),
	)
}

// SwarmInitiativeClient is the seam to swarm-manager. Plain HTTP via
// api-core/discovery, mirroring the agent-manager client pattern.
type SwarmInitiativeClient struct {
	httpClient  *http.Client
	testBaseURL string // tests only; empty in production
}

// NewSwarmInitiativeClient creates a new swarm-manager initiative client.
func NewSwarmInitiativeClient(timeout time.Duration) *SwarmInitiativeClient {
	return &SwarmInitiativeClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *SwarmInitiativeClient) resolveBaseURL(ctx context.Context, scenario string) (string, error) {
	if c.testBaseURL != "" {
		return c.testBaseURL, nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("resolve %s url: %w", scenario, err)
	}
	return url, nil
}

// Create posts to scenario's POST /api/v1/initiatives endpoint and returns
// the created initiative's name (or the canonical conflict / network error).
// Errors include the response status so callers can surface 409 distinctly.
func (c *SwarmInitiativeClient) Create(ctx context.Context, scenario string, req initiativeCreateRequest) (string, error) {
	baseURL, err := c.resolveBaseURL(ctx, scenario)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal create request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/initiatives", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call %s: %w", scenario, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var out initiativeCreateResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return req.Name, nil // success but unparseable body; return requested name
		}
		if out.Name != "" {
			return out.Name, nil
		}
		return req.Name, nil
	}
	// Try to surface the structured error message.
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(respBody, &errResp)
	msg := errResp.Message
	if msg == "" {
		msg = errResp.Error
	}
	if msg == "" {
		msg = strings.TrimSpace(string(respBody))
	}
	if msg == "" {
		msg = fmt.Sprintf("status %d", resp.StatusCode)
	}
	return "", fmt.Errorf("%s create returned %d: %s", scenario, resp.StatusCode, msg)
}
