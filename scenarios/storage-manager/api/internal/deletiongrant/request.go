package deletiongrant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

const (
	// Source and Kind compose the decision event type the resolver claims.
	Source          = "storage-manager"
	Kind            = "deletion-grant"
	defaultGrantTTL = 7 * 24 * time.Hour
	maxGrantTTL     = 30 * 24 * time.Hour

	extraRequestedTTL = "requested_ttl_seconds"
	// correlationAskID carries the producer's own ask id in Facts.Correlation so
	// the resolution (which arrives keyed by notification-hub's UUID, not ours)
	// can be mapped back to the stored ask — mirroring how deployment-manager
	// rebuilds its correlation key from resolution.Correlation.
	correlationAskID = "grant_ask_id"
)

// DecisionRaiser is the seam over operatordecision.Raiser so request building is
// unit-testable without the substrate's storage/publisher wiring.
type DecisionRaiser interface {
	Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error)
}

// Request raises an operator decision asking to delete paths for up to ttl.
// raised=true means the ask is outstanding (new or already pending), so the
// caller answers "pending"; the decision never grants authority — only an
// approved resolution does. The correlation id is stable for a given path set,
// so a retry reuses the one pending ask instead of flooding.
func Request(ctx context.Context, raiser DecisionRaiser, paths []string, requestedBy, raisedByRunID, webConsoleSessionID, reason string, ttl time.Duration) (string, bool, error) {
	return RequestProtected(ctx, raiser, nil, paths, requestedBy, raisedByRunID, webConsoleSessionID, reason, ttl)
}

// Protection names the declaration protecting path, if any, and whether path
// is that protected root itself. An error means protection is unknown, and the
// request is refused rather than asked without the warning.
type Protection func(path string) (source, root string, isRoot bool, err error)

// RequestProtected is Request with the host's declarations consulted: a path
// inside owner-protected data is flagged in the ask so the operator approves
// knowingly, and a path that is a protected root itself is refused, since no
// grant can open it.
func RequestProtected(ctx context.Context, raiser DecisionRaiser, protect Protection, paths []string, requestedBy, raisedByRunID, webConsoleSessionID, reason string, ttl time.Duration) (string, bool, error) {
	paths = normalizePaths(paths)
	if len(paths) == 0 {
		return "", false, errors.New("at least one absolute path is required")
	}
	protected := map[string]string{}
	if protect != nil {
		for _, p := range paths {
			source, root, isRoot, err := protect(p)
			switch {
			case err != nil:
				return "", false, fmt.Errorf("cannot tell whether %s is protected data: %w", p, err)
			case isRoot:
				return "", false, fmt.Errorf("%s is the root %s declares protected; a grant cannot open it, so change that declaration instead", p, source)
			case source != "":
				protected[p] = fmt.Sprintf("inside %s, which %s protects", root, source)
			}
		}
	}
	if ttl <= 0 {
		ttl = defaultGrantTTL
	}
	if ttl > maxGrantTTL {
		ttl = maxGrantTTL
	}
	id := operatordecision.CorrelationID(Kind, paths...)
	pathsJSON, err := json.Marshal(paths)
	if err != nil {
		return "", false, err
	}
	ask := operatordecision.PendingAsk{
		ID:      id,
		Subject: subjectFor(requestedBy),
		Extra: map[string]string{
			extraPaths:        string(pathsJSON),
			extraRequestedBy:  requestedBy,
			extraReason:       reason,
			extraRequestedTTL: strconv.FormatInt(int64(ttl/time.Second), 10),
		},
	}
	raised, err := raiser.Raise(ctx, ask, buildFacts(id, paths, protected, requestedBy, raisedByRunID, webConsoleSessionID, reason, ttl))
	return id, raised, err
}

// subjectFor keeps the flood cap per requesting agent, falling back to a shared
// subject when the caller is anonymous so a missing identity cannot evade it.
func subjectFor(requestedBy string) string {
	if s := strings.TrimSpace(requestedBy); s != "" {
		return s
	}
	return "storage-manager:deletion"
}

// normalizePaths cleans, absolutises-by-cleaning, de-duplicates, and sorts paths
// so the correlation id is stable regardless of request order or duplicates.
func normalizePaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || !filepath.IsAbs(p) {
			continue
		}
		p = filepath.Clean(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func buildFacts(askID string, paths []string, protected map[string]string, requestedBy, raisedByRunID, webConsoleSessionID, reason string, ttl time.Duration) operatordecision.Facts {
	note := strings.Builder{}
	fmt.Fprintf(&note, "Requesting agent: %s\n", requestedBy)
	if strings.TrimSpace(reason) != "" {
		fmt.Fprintf(&note, "Reason: %s\n", reason)
	}
	fmt.Fprintf(&note, "Grant validity if approved: %s\n", ttl)
	if len(protected) > 0 {
		fmt.Fprintf(&note, "WARNING: %d path(s) are owner-protected durable data; approving overrides that protection for these exact paths only.\n", len(protected))
	}
	note.WriteString("Paths:\n")
	for _, p := range paths {
		if where, ok := protected[p]; ok {
			fmt.Fprintf(&note, "  - %s (PROTECTED: %s)\n", p, where)
			continue
		}
		fmt.Fprintf(&note, "  - %s\n", p)
	}
	noteText := note.String()
	if len(noteText) > operatordecision.MaxQuestion {
		noteText = noteText[:operatordecision.MaxQuestion]
	}
	summary := fmt.Sprintf("%d path(s)", len(paths))
	if len(paths) == 1 {
		summary = paths[0]
	}
	correlation := map[string]string{
		correlationAskID: askID,
		extraRequestedBy: requestedBy,
		// Optional park companion: the agent, if it parks, parks on the
		// operator-decision producer with this key (the decision id).
		operatordecision.CorrelationAgentAwaitKey: askID,
	}
	// The trusted requester run id (from agent-provenance) enables agent-manager
	// to deliver the answer back to the agent automatically — parked or running.
	// Omitted for a non-agent caller, which simply gets no tie-back.
	if strings.TrimSpace(raisedByRunID) != "" {
		correlation[operatordecision.CorrelationRaisedByRunID] = raisedByRunID
	}
	// The stable web-console session id is the fallback delivery handle for a
	// web-console-hosted agent — it does not expire like the identity token, so
	// agent-manager can still reach a long-lived interactive session whose token
	// has aged out. The agent supplies its own session id.
	if strings.TrimSpace(webConsoleSessionID) != "" {
		correlation[operatordecision.CorrelationWebConsoleSessionID] = webConsoleSessionID
	}
	question := fmt.Sprintf("An agent requests authority to delete %s. Approve a time-boxed deletion grant?", summary)
	if len(protected) > 0 {
		question = fmt.Sprintf("An agent requests authority to delete %s, including owner-protected data. Approve a time-boxed deletion grant?", summary)
	}
	return operatordecision.Facts{
		Subject:  fmt.Sprintf("Approve agent deletion of %s?", summary),
		Question: question,
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve deletion"},
			{Key: "deny", Label: "Do not delete"},
		},
		// Deletion is irreversible: recommend deny, never auto-apply at a deadline.
		Recommended: "deny",
		Reversible:  false,
		Urgency:     "normal",
		Note:        noteText,
		Correlation: correlation,
	}
}
