// Package agentaccess exposes concise CLI discovery and qualification for
// harnesses that borrow an existing BAS-owned session.
package agentaccess

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
	"github.com/vrooli/cli-core/cliapp"
)

const (
	identityTokenEnv   = "VROOLI_AGENT_IDENTITY_TOKEN"
	grantReceiptEnv    = "VROOLI_BROWSER_GRANT_RECEIPT"
	consentReceiptEnv  = "BAS_BROWSER_CONSENT_RECEIPT"
	approvalReceiptEnv = "BAS_BROWSER_APPROVAL_RECEIPT"
	maxRequestBytes    = 1 << 20
)

type Harness struct {
	ID               string   `json:"id"`
	Version          string   `json:"version"`
	Summary          string   `json:"summary"`
	ObservationModes []string `json:"observation_modes"`
	ActionVocabulary []string `json:"action_vocabulary"`
	EffectClasses    []string `json:"effect_classes"`
	ReplayFidelity   string   `json:"replay_fidelity"`
	InferenceRoles   []string `json:"inference_roles"`
	Upstream         string   `json:"upstream,omitempty"`
	PinnedVersion    string   `json:"pinned_version,omitempty"`
}

type Qualification struct {
	AdapterID string   `json:"adapter_id"`
	Outcome   string   `json:"outcome"`
	CheckedAt string   `json:"checked_at,omitempty"`
	Evidence  []string `json:"evidence,omitempty"`
	Limits    []string `json:"limits,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type openInput struct {
	TabID          string `json:"tab_id,omitempty"`
	Backend        string `json:"backend,omitempty"`
	StartURL       string `json:"start_url,omitempty"`
	ReuseMode      string `json:"reuse_mode,omitempty"`
	ViewportWidth  int    `json:"viewport_width,omitempty"`
	ViewportHeight int    `json:"viewport_height,omitempty"`
}
type attachInput struct {
	PermissionRef string `json:"permission_ref"`
	TabID         string `json:"tab_id,omitempty"`
}
type observeInput struct {
	PageID string `json:"page_id,omitempty"`
}
type actInput struct {
	Instruction json.RawMessage `json:"instruction"`
	TargetURL   string          `json:"target_url,omitempty"`
}
type taskInput struct {
	Operation string `json:"operation"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	URL       string `json:"url,omitempty"`
}
type promoteInput struct {
	ProjectID   string            `json:"project_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Assertions  []json.RawMessage `json:"assertions"`
}
type draftTraceInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Assertions  []json.RawMessage `json:"assertions"`
}
type openWire struct {
	openInput
	ConsentReceipt string `json:"consent_receipt,omitempty"`
}
type attachWire struct {
	attachInput
	ConsentReceipt  string `json:"consent_receipt,omitempty"`
	ApprovalReceipt string `json:"approval_receipt,omitempty"`
}
type actWire struct {
	actInput
	ApprovalReceipt string `json:"approval_receipt,omitempty"`
}

// Commands returns the `browser` command group. It only discovers adapters or
// operates on BAS-owned sessions and never launches a browser locally.
func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Agent browser access", Commands: []cliapp.Command{{
		Name: "browser", NeedsAPI: true,
		Description: "Open, observe, act, run a common typed task, attach, trace, promote, or close a BAS-owned browser session; discover harnesses",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "operation", Required: true, Description: "open, observe, act, task, attach, close, trace, draft-trace, promote-trace, replay-flow, harnesses, or qualify"}},
			Flags: []cliapp.Flag{
				{Name: "session", Description: "Existing BAS browser session ID (required for every operation except open and harnesses)"},
				{Name: "adapter", Description: "Harness adapter ID (required for qualify)"},
				{Name: "request-file", Description: "JSON request file for open, attach, observe, act, task, draft-trace, or promote-trace (maximum 1 MiB)"},
				{Name: "json", Bool: true, Description: "Emit JSON (the default)"},
			},
		},
		RunCtx: func(rc cliapp.RunContext) error {
			operation := rc.Positional("operation")
			switch operation {
			case "harnesses":
				if strings.TrimSpace(rc.Flag("session")) != "" || strings.TrimSpace(rc.Flag("adapter")) != "" {
					return errors.New("--session and --adapter are only valid with browser qualify")
				}
				return listHarnesses(ctx, rc.Stdout())
			case "qualify":
				return qualify(ctx, strings.TrimSpace(rc.Flag("session")), strings.TrimSpace(rc.Flag("adapter")), rc.Stdout())
			case "open":
				return open(ctx, rc.Flag("request-file"), rc.Stdout())
			case "attach":
				return attach(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "observe":
				return observe(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "act":
				return act(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "task":
				return runTask(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "trace":
				return trace(ctx, strings.TrimSpace(rc.Flag("session")), rc.Stdout())
			case "draft-trace":
				return draftTrace(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "promote-trace":
				return promote(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "replay-flow":
				return replayFlow(ctx, strings.TrimSpace(rc.Flag("session")), rc.Flag("request-file"), rc.Stdout())
			case "close":
				return closeSession(ctx, strings.TrimSpace(rc.Flag("session")))
			default:
				return fmt.Errorf("unsupported browser operation %q", operation)
			}
		},
	}}}
}

func runTask(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser task requires a safe existing --session ID")
	}
	var input taskInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if strings.TrimSpace(input.Operation) == "" {
		return errors.New("browser task request requires operation")
	}
	return doJSON(ctx, "POST", sessionPath(sessionID, "task"), input, out)
}

func open(ctx *appctx.Context, requestFile string, out io.Writer) error {
	var input openInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	wire := openWire{openInput: input, ConsentReceipt: strings.TrimSpace(os.Getenv(consentReceiptEnv))}
	return doJSON(ctx, "POST", "/browser/sessions", wire, out)
}

func attach(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser attach requires a safe existing --session ID")
	}
	var input attachInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if strings.TrimSpace(input.PermissionRef) == "" {
		return errors.New("browser attach request requires permission_ref")
	}
	wire := attachWire{attachInput: input, ConsentReceipt: strings.TrimSpace(os.Getenv(consentReceiptEnv)), ApprovalReceipt: strings.TrimSpace(os.Getenv(approvalReceiptEnv))}
	return doJSON(ctx, "POST", sessionPath(sessionID, "attach"), wire, out)
}

func observe(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser observe requires a safe existing --session ID")
	}
	var input observeInput
	if requestFile != "" {
		if err := readStrictRequest(requestFile, &input); err != nil {
			return err
		}
	}
	return doJSON(ctx, "POST", sessionPath(sessionID, "observe"), input, out)
}

func act(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser act requires a safe existing --session ID")
	}
	var input actInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if len(input.Instruction) == 0 {
		return errors.New("browser act request requires typed instruction")
	}
	wire := actWire{actInput: input, ApprovalReceipt: strings.TrimSpace(os.Getenv(approvalReceiptEnv))}
	return doJSON(ctx, "POST", sessionPath(sessionID, "act"), wire, out)
}

func trace(ctx *appctx.Context, sessionID string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser trace requires a safe existing --session ID")
	}
	return doJSON(ctx, "GET", sessionPath(sessionID, "trace"), nil, out)
}

func draftTrace(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser draft-trace requires a safe existing --session ID")
	}
	var input draftTraceInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if strings.TrimSpace(input.Name) == "" || len(input.Assertions) == 0 {
		return errors.New("browser draft-trace requires name and at least one typed assertion")
	}
	return doJSON(ctx, "POST", sessionPath(sessionID, "trace/draft"), input, out)
}

func promote(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser promote-trace requires a safe existing --session ID")
	}
	var input promoteInput
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.Name) == "" || len(input.Assertions) == 0 {
		return errors.New("browser promote-trace requires project_id, name, and at least one typed assertion")
	}
	return doJSON(ctx, "POST", sessionPath(sessionID, "trace/promote"), input, out)
}

func replayFlow(ctx *appctx.Context, sessionID, requestFile string, out io.Writer) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser replay-flow requires a safe existing --session ID")
	}
	var input struct {
		WorkflowID      string            `json:"workflow_id"`
		WorkflowVersion int32             `json:"workflow_version"`
		Parameters      map[string]string `json:"parameters,omitempty"`
	}
	if err := readStrictRequest(requestFile, &input); err != nil {
		return err
	}
	if !safePathSegment(input.WorkflowID) || input.WorkflowVersion <= 0 {
		return errors.New("browser replay-flow requires a safe workflow_id and positive workflow_version")
	}
	return doJSON(ctx, "POST", sessionPath(sessionID, "workflows/"+input.WorkflowID+"/replay"), input, out)
}

func closeSession(ctx *appctx.Context, sessionID string) error {
	if !safePathSegment(sessionID) {
		return errors.New("browser close requires a safe existing --session ID")
	}
	status, body, err := request(ctx, "POST", sessionPath(sessionID, "close"), nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return responseError(status, body)
	}
	return nil
}

func doJSON(ctx *appctx.Context, method, path string, payload any, out io.Writer) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode browser request: %w", err)
		}
	}
	if len(body) > maxRequestBytes {
		return errors.New("browser request exceeds 1 MiB limit")
	}
	status, response, err := request(ctx, method, path, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return responseError(status, response)
	}
	if len(response) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(response, &value); err != nil {
		return fmt.Errorf("decode BAS browser response: %w", err)
	}
	return printJSON(out, redactAuthorityFields(value))
}

func redactAuthorityFields(value any) any {
	blocked := map[string]struct{}{
		"permissionref": {}, "permissionprofile": {}, "profile": {}, "profileref": {}, "profileid": {},
		"identityprofile": {}, "identityevidence": {}, "identitytoken": {}, "grantreceipt": {},
		"browsergrantreceipt": {}, "credential": {}, "credentialid": {}, "credentials": {},
	}
	switch current := value.(type) {
	case map[string]any:
		for key, nested := range current {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if _, denied := blocked[normalized]; denied {
				delete(current, key)
				continue
			}
			current[key] = redactAuthorityFields(nested)
		}
		return current
	case []any:
		for index := range current {
			current[index] = redactAuthorityFields(current[index])
		}
		return current
	default:
		return value
	}
}

func readStrictRequest(path string, target any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("--request-file is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open browser request file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read browser request file: %w", err)
	}
	if len(data) == 0 || len(data) > maxRequestBytes {
		return errors.New("browser request file must contain 1 byte to 1 MiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode browser request file: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("browser request file must contain exactly one JSON object")
	}
	return nil
}

func sessionPath(sessionID, suffix string) string {
	return "/browser/sessions/" + url.PathEscape(sessionID) + "/" + suffix
}

func listHarnesses(ctx *appctx.Context, out io.Writer) error {
	status, body, err := request(ctx, "GET", "/browser/harnesses", nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return responseError(status, body)
	}
	var result []Harness
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode BAS harness registry: %w", err)
	}
	return printJSON(out, result)
}

func qualify(ctx *appctx.Context, sessionID, adapterID string, out io.Writer) error {
	if sessionID == "" || adapterID == "" {
		return errors.New("browser qualify requires --session and --adapter")
	}
	if !safePathSegment(sessionID) || !safePathSegment(adapterID) {
		return errors.New("session and adapter IDs may contain only letters, numbers, period, underscore, and hyphen")
	}
	path := "/browser/sessions/" + url.PathEscape(sessionID) + "/harnesses/" + url.PathEscape(adapterID) + "/qualify"
	status, body, err := request(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return responseError(status, body)
	}
	var result Qualification
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode BAS harness qualification: %w", err)
	}
	return printJSON(out, result)
}

func request(ctx *appctx.Context, method, path string, body []byte) (int, []byte, error) {
	identityToken := strings.TrimSpace(os.Getenv(identityTokenEnv))
	grantReceipt := strings.TrimSpace(os.Getenv(grantReceiptEnv))
	if identityToken == "" || grantReceipt == "" {
		return 0, nil, fmt.Errorf("Agent Manager browser context is unavailable; require %s and %s", identityTokenEnv, grantReceiptEnv)
	}
	return api.Do(ctx, method, path, nil, body, map[string]string{
		"X-Agent-Identity-Token":         identityToken,
		"X-Vrooli-Browser-Grant-Receipt": grantReceipt,
	})
}

func printJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func safePathSegment(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return value != "" && value != "." && value != ".."
}

func responseError(status int, body []byte) error {
	_ = body // Response details can contain target page text; keep errors opaque.
	return fmt.Errorf("BAS browser access returned HTTP %d", status)
}
