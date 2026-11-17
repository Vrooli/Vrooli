package deletiongrant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// AskWindow is how long an unanswered deletion decision stays open before the
// reconciler closes it.
const AskWindow = 7 * 24 * time.Hour

// RequestInput is one agent's request to delete paths.
type RequestInput struct {
	Paths  []string
	Reason string
	// RequestedBy is the caller's subject (the flood cap is per subject);
	// RaisedByRunID and WebConsoleSessionID are the answer tie-back handles.
	RequestedBy, RaisedByRunID, WebConsoleSessionID string
	TTL                                             time.Duration

	now  time.Time // tests pin the clock and home folder the page is written from
	home string
}

// Request raises an operator decision asking to delete paths for up to TTL.
// declare says what the host's declarations hold at each path: a path inside
// owner-protected data is flagged so the operator approves knowingly, and a
// path that is a protected root itself is refused, since no grant can open it.
// raised=true means the ask is outstanding (new or already pending), so the
// caller answers "pending"; the decision never grants authority — only an
// approved resolution does. The correlation id is stable for a given path set,
// so a retry reuses the one pending ask instead of flooding.
func Request(ctx context.Context, raiser DecisionRaiser, declare func(string) Declaration, in RequestInput) (string, bool, error) {
	paths := normalizePaths(in.Paths)
	if len(paths) == 0 {
		return "", false, errors.New("at least one absolute path is required")
	}
	if declare != nil {
		for _, p := range paths {
			if d := declare(p); d.IsRoot {
				return "", false, fmt.Errorf("%s is the root %s declares protected; a grant cannot open it, so change that declaration instead", p, d.Source)
			}
		}
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = defaultGrantTTL
	}
	if ttl > maxGrantTTL {
		ttl = maxGrantTTL
	}
	in.TTL = ttl
	if in.now.IsZero() {
		in.now = time.Now()
	}
	if in.home == "" {
		in.home, _ = os.UserHomeDir()
	}
	id := operatordecision.CorrelationID(Kind, paths...)
	pathsJSON, err := json.Marshal(paths)
	if err != nil {
		return "", false, err
	}
	ask := operatordecision.PendingAsk{
		ID:      id,
		Subject: subjectFor(in.RequestedBy),
		Extra: map[string]string{
			extraPaths:        string(pathsJSON),
			extraRequestedBy:  in.RequestedBy,
			extraReason:       in.Reason,
			extraRequestedTTL: strconv.FormatInt(int64(ttl/time.Second), 10),
			extraSubjectKey:   id,
		},
	}
	items := inspect(paths, declare, defaultBudget(in.now))
	raised, err := raiser.Raise(ctx, ask, buildFacts(id, items, in))
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

// KeyedRaiser raises through Inner but keeps an ask that was raised before
// subject keys unkeyed while it is still outstanding: its Hub decision has no
// subject, so a keyed re-send would open a second decision beside it. Such an
// ask is closed by its request event id when it lapses or its paths are gone;
// a request after it lapsed raises a keyed ask.
type KeyedRaiser struct {
	Inner DecisionRaiser
	Store *Store
	Now   func() time.Time
}

// Raise implements DecisionRaiser.
func (k KeyedRaiser) Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error) {
	existing, err := k.Store.Get(ctx, ask.ID)
	if err != nil {
		return false, err
	}
	now := time.Now()
	if k.Now != nil {
		now = k.Now()
	}
	if existing != nil && existing.Outstanding(now) && existing.Extra[extraSubjectKey] == "" {
		facts.SubjectKey = ""
	}
	return k.Inner.Raise(ctx, ask, facts)
}
