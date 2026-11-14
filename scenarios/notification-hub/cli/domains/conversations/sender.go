package conversations

// The sender an ask carries when the command did not say with --from: a human
// name for the agent's own session, so the operator can tell agents apart
// without every agent having to name itself.
//
// The Hub cannot resolve this itself (it holds no Web Console credential), so
// the CLI does, on the agent's machine, from up to three reads:
//
//	fresh identity credential -> run id -> the run -> its Web Console session -> that session's label
//
// The name is, in order, the Web Console session's label, then the run's own
// label when someone set it by hand (label_source "manual"). A generic session
// label ("This machine") and the opaque labels Agent Manager gives a run itself
// (label_source "harness" or "placeholder": "claude-code session <hash>") are no
// name, and then the Hub names the sender from what it verified.
//
// It is a convenience and never a requirement. It runs at most once per
// process, within senderBudget in total, and any failure at all (no identity,
// Agent Manager or Web Console unreachable, a run with no session, a session
// with no label, a timeout) means the ask is sent without a default and the Hub
// names the sender itself. It must never fail or noticeably delay an ask.

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

// senderBudget bounds the whole resolution, not each read. It is a variable only
// so a test of the overrun does not wait it out.
var senderBudget = 1500 * time.Millisecond

// senderSource resolves the label of the calling agent's session. It reports
// an error for anything it cannot resolve; the caller discards the reason.
type senderSource func(ctx context.Context) (string, error)

// defaultSender is the sender for an ask that did not name one, or "" when
// there is none. The resolution runs once per process: a second ask in the same
// process (a retry, a wait loop) sees the same answer, so one process never
// sends the same terms with and without a default.
func (s *service) defaultSender() string {
	s.senderOnce.Do(func() {
		if s.senderSource == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), senderBudget)
		defer cancel()
		type result struct {
			label string
			err   error
		}
		done := make(chan result, 1)
		// The resolution may block on a read that ignores the context (port
		// discovery shells out), so the budget is enforced here and an overrun
		// is abandoned rather than waited for.
		go func() {
			label, err := s.senderSource(ctx)
			done <- result{label, err}
		}()
		select {
		case r := <-done:
			if r.err == nil {
				s.sender = tidySender(r.label)
			}
		case <-ctx.Done():
		}
	})
	return s.sender
}

// tidySender turns a resolved label into a sender the Hub accepts: blanks
// collapsed, control characters dropped, at most maxFromRunes characters cut on
// a character boundary. Unlike --from, which a person typed and which is refused
// when wrong, a label nobody typed is repaired.
func tidySender(label string) string {
	label = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(label, ""))
	label = strings.Join(strings.Fields(label), " ")
	if utf8.RuneCountInString(label) > maxFromRunes {
		label = strings.TrimSpace(string([]rune(label)[:maxFromRunes]))
	}
	return label
}

// senderOfThisRun is the production senderSource.
func senderOfThisRun(ctx context.Context) (string, error) {
	// The credential comes from the live authority, as every other request does:
	// the token inherited in the environment expires in a long session and then
	// fails verification. The run id comes only from the verified credential;
	// VROOLI_AGENT_MANAGER_RUN_ID is retired and never attributes anything.
	credential, err := cliutil.ResolveAgentCredential(ctx)
	if err != nil || credential == "" {
		return "", errors.New("no agent identity")
	}
	verified, err := cliutil.IdentityEnv{Token: credential}.VerifyIdentity()
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.RunID == "" {
		return "", errors.New("identity not verified")
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	sessions := sessionsconnect.NewSessionsServiceClient(http.DefaultClient, scenarioBase("web-console", "WEB_CONSOLE_API_BASE", "WEB_CONSOLE_API_URL"))
	return senderOfRun(ctx, verified.Claims.RunID, readRun, sessions.Get)
}

// scenarioBase is the API base of another scenario on this machine: the named
// environment variables, then the scenario's port.
func scenarioBase(scenario string, envVars ...string) string {
	return cliutil.DetermineAPIBase(cliutil.APIBaseOptions{
		EnvVars:      envVars,
		PortDetector: cliutil.DetectPortFromVrooli(scenario, "API_PORT"),
	})
}

// runRecord is the part of an Agent Manager run that names who it is.
type runRecord struct {
	// SessionID is the Web Console session hosting the run; empty when none does.
	SessionID string
	// Label and LabelSource are the run's own label and who set it: "manual" for
	// a name a person or a launch path gave, "harness" or "placeholder" for one
	// Agent Manager made up from an id.
	Label, LabelSource string
}

type runReader func(ctx context.Context, runID string) (runRecord, error)

type sessionReader func(context.Context, *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error)

// readRun reads a run from Agent Manager's open run read, GET /api/v1/runs/{id}.
// It decodes the few fields it needs instead of using the generated Agent
// Manager client: that client pulls in protovalidate and genproto modules this
// CLI does not otherwise carry, and a new dependency for a few strings is not
// worth the governance it would need.
func readRun(ctx context.Context, runID string) (runRecord, error) {
	base := scenarioBase("agent-manager", "AGENT_MANAGER_API_BASE", "AGENT_MANAGER_API_URL")
	if base == "" {
		return runRecord{}, errors.New("agent-manager is not reachable")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/runs/"+url.PathEscape(runID), nil)
	if err != nil {
		return runRecord{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return runRecord{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return runRecord{}, fmt.Errorf("agent-manager answered %d", resp.StatusCode)
	}
	var body struct {
		Run struct {
			SessionID      string `json:"web_console_session_id"`
			SessionIDCamel string `json:"webConsoleSessionId"`
			Label          string `json:"label"`
			LabelSource    string `json:"label_source"`
			LabelSourceCml string `json:"labelSource"`
		} `json:"run"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return runRecord{}, err
	}
	run := body.Run
	return runRecord{
		SessionID:   cmp.Or(run.SessionID, run.SessionIDCamel),
		Label:       run.Label,
		LabelSource: cmp.Or(run.LabelSource, run.LabelSourceCml),
	}, nil
}

// senderOfRun names the sender for a run: its Web Console session's label, then
// its own label when it was set by hand. A read that fails only loses that
// source; the next one is still tried.
func senderOfRun(ctx context.Context, runID string, read runReader, readSession sessionReader) (string, error) {
	run, err := read(ctx, runID)
	if err != nil {
		return "", err
	}
	if run.SessionID != "" {
		if session, err := readSession(ctx, connect.NewRequest(&sessionsv1.GetRequest{Id: run.SessionID})); err == nil {
			if name := roleOfSessionLabel(session.Msg.GetSession().GetDisplayLabel()); name != "" {
				return name, nil
			}
		}
	}
	if run.LabelSource == "manual" {
		if name := roleOfSessionLabel(run.Label); name != "" {
			return name, nil
		}
	}
	return "", errors.New("the run has no label that names an agent")
}

// genericSessionLabels are labels every session of some kind gets, so they say
// nothing about who is asking: the Web Console's own name for a session opened
// from its UI.
var genericSessionLabels = []string{"This machine"}

// runDateSuffix is the run-specific tail of a programmatic label: a date, a
// time of day if there is one, and any counters after it ("-20261114",
// "-20261114T0401", "-20261113-2", "-20261112-r3").
var runDateSuffix = regexp.MustCompile(`-\d{8}(?:T\d{4,6}Z?)?(?:-[A-Za-z]?\d+)*$`)

// roleOfSessionLabel turns a session or run label into the name of the role that
// asks: the label tidied, with a generic label dropped and a trailing run date
// removed, so one role keeps one sender across days. The sender is a filter
// choice, and a name that changes daily would split one role's asks across as
// many choices. The rest of the label is kept as it is. It is "" when nothing
// that names an agent is left.
func roleOfSessionLabel(label string) string {
	label = tidySender(label)
	for _, generic := range genericSessionLabels {
		if strings.EqualFold(label, generic) {
			return ""
		}
	}
	return tidySender(runDateSuffix.ReplaceAllString(label, ""))
}
