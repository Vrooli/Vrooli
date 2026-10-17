package agentsession

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/agentscope"
)

// probeTimeout bounds every read-only owner query.
const probeTimeout = 5 * time.Second

// hostFacts is the live HostFacts.
type hostFacts struct {
	once      sync.Once
	slicePath string
	sliceErr  error
}

// NewHostFacts reads this host. Off Linux the process readers report
// unsupported and owners stay unknown.
func NewHostFacts() HostFacts { return &hostFacts{} }

func (h *hostFacts) Environment(pid int) (map[string]string, error) {
	return platformgo.ReadProcessEnvironment(pid)
}

func (h *hostFacts) Process(pid int) (ProcessFact, error) { return processFact(pid) }

func (h *hostFacts) ProcessName(pid int) (string, error) { return platformgo.ProcessName(pid) }

func (h *hostFacts) StartedAt(pid int) (time.Time, error) { return processStartedAt(pid) }

func (h *hostFacts) TTYActivity(tty string) (time.Time, error) { return ttyActivity(tty) }

func (h *hostFacts) ProcessMemory(pid int) (int64, int64, error) { return processMemory(pid) }

func (h *hostFacts) ScopeMembers(scopeUnit string) ([]int, error) {
	ref, err := h.scopeRef(scopeUnit)
	if err != nil {
		return nil, err
	}
	return platformgo.ScopeProcesses(ref)
}

// scopeRef resolves a session scope unit to its platform ref under the agent
// slice. The slice path is asked of the user manager once per process.
func (h *hostFacts) scopeRef(scopeUnit string) (platformgo.ScopeRef, error) {
	h.once.Do(func() { h.slicePath, h.sliceErr = platformgo.SliceCgroup(agentscope.Slice) })
	if h.sliceErr != nil {
		return platformgo.ScopeRef{}, h.sliceErr
	}
	return scopeRefUnder(h.slicePath, scopeUnit)
}

func scopeRefUnder(slicePath, scopeUnit string) (platformgo.ScopeRef, error) {
	if !agentscope.IsSessionScope(scopeUnit) || strings.ContainsAny(scopeUnit, "/\\") {
		return platformgo.ScopeRef{}, fmt.Errorf("agentsession: %q is not an agent session scope unit", scopeUnit)
	}
	ref := agentscope.Ref(strings.TrimRight(slicePath, "/") + "/" + scopeUnit)
	ref.Name = strings.TrimSuffix(scopeUnit, agentscope.ScopeSuffix)
	return ref, nil
}

// PlatformStopper stops a session through platform-go's containment seam:
// the systemd scope on Linux; ErrUnsupported elsewhere until a process-group
// (macOS) or Job Object (Windows) adapter is host-verified.
type PlatformStopper struct {
	host *hostFacts
}

// NewPlatformStopper binds the stopper to this host.
func NewPlatformStopper() PlatformStopper { return PlatformStopper{host: &hostFacts{}} }

// Stop ends the session's scope.
func (p PlatformStopper) Stop(_ context.Context, session Session) error {
	unit := session.ScopeUnit()
	if unit == "" {
		return fmt.Errorf("agentsession: session %s has no agent scope unit to stop", session.SessionID)
	}
	host := p.host
	if host == nil {
		host = &hostFacts{}
	}
	ref, err := host.scopeRef(unit)
	if err != nil {
		return err
	}
	return platformgo.StopScope(ref)
}

// LiveProbes binds the owner probes to this host's Agent Manager, web-console
// and process table. Every probe is read-only.
func LiveProbes(home string) Probes {
	stateDir, _ := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	client := &http.Client{Timeout: probeTimeout}
	agentManager, webConsole := resolveBase("agent-manager"), resolveBase("web-console")
	return Probes{
		RunStatus: func(ctx context.Context, runID string) (string, error) {
			return agentManagerRunStatus(ctx, client, agentManager, runID)
		},
		WebConsoleSession: func(ctx context.Context, stateRoot, sessionID string) (WebConsoleSession, error) {
			if !HostWebConsoleRoot(stateRoot, stateDir) {
				return WebConsoleSession{}, fmt.Errorf("session state root %q belongs to another web-console instance (desktop build or variant)", stateRoot)
			}
			return webConsoleGetSession(ctx, client, webConsole, sessionID)
		},
		ProcessStartTime:    processStartTime,
		DesktopAppProcesses: desktopAppProcesses,
	}
}

// resolveBase memoizes one scenario's base URL for the process.
func resolveBase(scenario string) func(ctx context.Context) (string, error) {
	var once sync.Once
	var base string
	var err error
	return func(ctx context.Context) (string, error) {
		once.Do(func() {
			resolveCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			base, err = discovery.ResolveScenarioURLDefault(resolveCtx, scenario)
			base = strings.TrimRight(base, "/")
		})
		return base, err
	}
}

// agentManagerRunStatus reads GET /api/v1/runs/{id}. A 404 is "not found"
// only when Agent Manager itself says so (NOT_FOUND_*): a 404 from a missing
// route must never read as a vanished run.
func agentManagerRunStatus(ctx context.Context, client *http.Client, base func(context.Context) (string, error), runID string) (string, error) {
	root, err := base(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, root+"/api/v1/runs/"+url.PathEscape(runID), nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	switch response.StatusCode {
	case http.StatusOK:
		var payload struct {
			Run struct {
				Status string `json:"status"`
			} `json:"run"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return "", fmt.Errorf("decode run: %w", err)
		}
		if strings.TrimSpace(payload.Run.Status) == "" {
			return "", errors.New("run has no status")
		}
		return payload.Run.Status, nil
	case http.StatusNotFound:
		var payload struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(body, &payload) == nil && strings.HasPrefix(strings.ToUpper(payload.Code), "NOT_FOUND") {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("agent-manager returned 404 without a not-found code")
	default:
		return "", fmt.Errorf("agent-manager returned HTTP %d", response.StatusCode)
	}
}

// webConsoleGetProcedure is web-console's read-only session lookup
// (Connect, JSON codec).
const webConsoleGetProcedure = "/vrooli.web_console.v1.sessions.SessionsService/Get"

// webConsoleGetSession asks web-console for a live session. Connect reports
// an unknown, archived or dismissed session as code not_found.
func webConsoleGetSession(ctx context.Context, client *http.Client, base func(context.Context) (string, error), sessionID string) (WebConsoleSession, error) {
	root, err := base(ctx)
	if err != nil {
		return WebConsoleSession{}, fmt.Errorf("resolve web-console: %w", err)
	}
	payload, err := json.Marshal(map[string]string{"id": sessionID})
	if err != nil {
		return WebConsoleSession{}, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(reqCtx, http.MethodPost, root+webConsoleGetProcedure, bytes.NewReader(payload))
	if err != nil {
		return WebConsoleSession{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	response, err := client.Do(request)
	if err != nil {
		return WebConsoleSession{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return WebConsoleSession{}, err
	}
	if response.StatusCode != http.StatusOK {
		var connectErr struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(body, &connectErr) == nil && connectErr.Code == "not_found" {
			return WebConsoleSession{}, ErrNotFound
		}
		return WebConsoleSession{}, fmt.Errorf("web-console returned HTTP %d", response.StatusCode)
	}
	var got struct {
		Session struct {
			ID     string `json:"id"`
			Origin string `json:"origin"`
			Owner  string `json:"owner"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return WebConsoleSession{}, fmt.Errorf("decode session: %w", err)
	}
	if got.Session.ID != "" && got.Session.ID != sessionID {
		return WebConsoleSession{}, fmt.Errorf("web-console answered for session %s, not %s", got.Session.ID, sessionID)
	}
	return WebConsoleSession{Origin: originName(got.Session.Origin), Owner: got.Session.Owner}, nil
}

// originName maps the wire enum to the store's spelling; an absent origin is
// UNSPECIFIED on the wire.
func originName(wire string) string {
	switch strings.ToUpper(strings.TrimSpace(wire)) {
	case "SESSION_ORIGIN_UI", "UI":
		return OriginUI
	case "SESSION_ORIGIN_PROGRAMMATIC", "PROGRAMMATIC":
		return OriginProgrammatic
	case "SESSION_ORIGIN_REMOTE", "REMOTE":
		return OriginRemote
	default:
		return OriginUnspecified
	}
}

// OwnerAtLaunch is the owner a launcher records for the session it is about
// to start: inferred from the launcher's own environment and terminal, and,
// for a bundled desktop app, bound to the app runtime's pid and start time so
// the session becomes orphaned when the app exits.
func OwnerAtLaunch(env map[string]string, self int, host HostFacts) Owner {
	tty := ""
	if host != nil {
		if fact, err := host.Process(self); err == nil {
			tty = fact.TTY
		}
	}
	owner := Infer(env, tty)
	owner.Source = SourceLaunch
	owner.Evidence.ReadFromPID = self
	if owner.Kind == KindDesktopApp && host != nil {
		if root, ok := DesktopAppRoot(self, owner.Ref, host.Process, host.Environment); ok {
			owner.Evidence.AppPID, owner.Evidence.AppStartTime = root.PID, root.StartTime
		}
	}
	return owner
}

// EnvironMap turns KEY=VALUE pairs into a map; the last value wins.
func EnvironMap(environ []string) map[string]string {
	out := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key != "" {
			out[key] = value
		}
	}
	return out
}
