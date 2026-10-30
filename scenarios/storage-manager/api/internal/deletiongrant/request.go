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
	"unicode/utf8"

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

// ProtectionSource reads the host's declarations once and answers every path
// of one request from that read. Reading them per path made a large request
// take minutes.
type ProtectionSource func() (Protection, error)

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
	summary := pathSummary(paths)
	big := strings.Builder{}
	if strings.TrimSpace(reason) != "" {
		fmt.Fprintf(&big, "%s\n\n", strings.TrimSpace(reason))
	}
	fmt.Fprintf(&big, "Requesting agent: %s\nGrant validity if approved: %s. The grant opens exactly the %d listed path(s), nothing beside or above them.\n", requestedBy, ttl, len(paths))
	if len(protected) > 0 {
		fmt.Fprintf(&big, "WARNING: %d path(s) are owner-protected durable data; approving overrides that protection for these exact paths only.\n", len(protected))
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
		Subject:  subjectLine(paths),
		Question: truncateBytes(question, operatordecision.MaxQuestion),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve deletion"},
			{Key: "deny", Label: "Do not delete"},
		},
		// Deletion is irreversible: recommend deny, never auto-apply at a deadline.
		Recommended:      "deny",
		Reversible:       false,
		Urgency:          "normal",
		BigPicture:       truncateRunes(big.String(), operatordecision.MaxBigPictureCodePoints),
		TechnicalDetails: pathListing(paths, protected, operatordecision.MaxTechnicalDetailsCodePoints),
		Correlation:      correlation,
	}
}

// pathSummary names the request in a few words: the one path, or the count and
// the deepest directory every path shares, so a large ask reads at a glance.
func pathSummary(paths []string) string {
	if len(paths) == 1 {
		return paths[0]
	}
	parent := commonParent(paths)
	if parent == "" || parent == string(filepath.Separator) {
		return fmt.Sprintf("%d paths", len(paths))
	}
	return fmt.Sprintf("%d paths under %s", len(paths), parent)
}

// commonParent is the deepest directory that contains every path.
func commonParent(paths []string) string {
	parent := filepath.Dir(paths[0])
	for _, p := range paths[1:] {
		for parent != string(filepath.Separator) && parent != "." && !strings.HasPrefix(p, parent+string(filepath.Separator)) {
			parent = filepath.Dir(parent)
		}
	}
	return parent
}

// pathListing is the reviewer's view of the exact path set within limit code
// points: one line per containing directory with its count, then every path
// (protected ones marked) until the budget runs out, then how many are not
// shown. The full set is stored on the ask; the grant opens only that set.
func pathListing(paths []string, protected map[string]string, limit int) string {
	byDir := map[string]int{}
	for _, p := range paths {
		byDir[filepath.Dir(p)]++
	}
	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	out := strings.Builder{}
	fmt.Fprintf(&out, "%d path(s) in %d directory(ies):\n", len(paths), len(dirs))
	for _, dir := range dirs {
		fmt.Fprintf(&out, "  %s/ — %d\n", dir, byDir[dir])
	}
	out.WriteString("\nPaths:\n")
	const reserve = 120 // room for the "not shown" line
	used := utf8.RuneCountInString(out.String())
	for i, p := range paths {
		line := "  - " + p + "\n"
		if where, ok := protected[p]; ok {
			line = fmt.Sprintf("  - %s (PROTECTED: %s)\n", p, where)
		}
		n := utf8.RuneCountInString(line)
		if used+n > limit-reserve {
			fmt.Fprintf(&out, "  … and %d more not shown (%d protected among all paths).\n", len(paths)-i, len(protected))
			break
		}
		out.WriteString(line)
		used += n
	}
	return truncateRunes(out.String(), limit)
}

func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	r := []rune(s)
	return string(r[:limit])
}

// truncateBytes cuts s to at most limit bytes on a rune boundary.
func truncateBytes(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit]
}

// subjectLine is the inbox title: the one path, or just the count, since a long
// location would be cut off there; the question names where the paths are.
func subjectLine(paths []string) string {
	if len(paths) == 1 {
		return fmt.Sprintf("Approve agent deletion of %s?", paths[0])
	}
	return fmt.Sprintf("Approve agent deletion of %d paths?", len(paths))
}
