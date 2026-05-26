package eligibility

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"test-genie/internal/orchestrator/workspace"
)

// Rule IDs the eligibility decision keys off of.
const (
	RuleRoutedDrivers         = "routed_database_drivers"
	RuleRoutedHandleCapture   = "routed_database_handle_capture"
	RuleDatabaseBackoff       = "database_backoff"
	defaultEligibilityLimit   = 100
	disqualifyHighSevRoutedDB = true
)

// Eligibility is the outcome of a routing-eligibility check.
type Eligibility struct {
	// Routed is true when the scenario qualifies for the in-place routed
	// e2e path. False means the playbooks phase must take the fallback
	// (restart) path.
	Routed bool

	// Violations is the subset of the scan's TopViolations relevant to the
	// routing decision. Empty when Routed is true.
	Violations []ViolationExcerpt

	// RuleAssertion is non-nil when the auditor scan did not include one or
	// more of the three routing rules (rule is unregistered or disabled).
	// A non-nil assertion disqualifies the scenario even when there are no
	// observed violations: an unchecked rule is not the same as a passing
	// one.
	RuleAssertion *RuleAssertion

	// Summary is the full scan summary the routing decision was made from.
	// Both phase_standards and phase_playbooks consume it for their own
	// downstream reporting.
	Summary *ViolationSummary
}

// Checker fetches and caches per-scenario eligibility for the lifetime of a
// test-genie run.
type Checker struct {
	limit int

	mu    sync.Mutex
	cache map[string]Eligibility
}

// NewChecker returns a Checker that requests up to `summaryLimit`
// top-violations per scan; pass 0 to use the default.
func NewChecker(summaryLimit int) *Checker {
	if summaryLimit <= 0 {
		summaryLimit = defaultEligibilityLimit
	}
	return &Checker{
		limit: summaryLimit,
		cache: map[string]Eligibility{},
	}
}

// Check returns the eligibility of `scenario` for the routed path. The
// underlying auditor scan is shared with phase_standards (callers may also
// reach for `Summary` on the returned Eligibility to avoid running a second
// scan).
func (c *Checker) Check(ctx context.Context, scenario string, mapping workspace.Mapping) (Eligibility, error) {
	c.mu.Lock()
	if cached, ok := c.cache[scenario]; ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	baseURL, err := ResolveBaseURL(ctx)
	if err != nil {
		return Eligibility{}, err
	}

	summary, err := FetchSummary(ctx, nil, baseURL, scenario, mapping, c.limit)
	if err != nil {
		return Eligibility{}, err
	}

	elig := decide(summary)

	// Independent of violations, verify that the auditor actually ran the
	// three routing-rule checks. A scan that skipped them cannot certify
	// eligibility — treat as disqualifying.
	registered, regErr := FetchRegisteredRules(ctx, baseURL)
	if regErr != nil {
		return Eligibility{}, fmt.Errorf("scenario-auditor rule registry: %w", regErr)
	}
	if assertion := AssertRulesObserved(registered, RuleRoutedDrivers, RuleRoutedHandleCapture, RuleDatabaseBackoff); assertion != nil {
		elig.Routed = false
		elig.RuleAssertion = assertion
	}

	c.mu.Lock()
	c.cache[scenario] = elig
	c.mu.Unlock()
	return elig, nil
}

// Invalidate drops the cached eligibility for a single scenario so the next
// Check re-fetches. Used by the playbooks-phase claim defer so a successive
// run in the same test-genie process picks up code fixes made between runs.
func (c *Checker) Invalidate(scenario string) {
	c.mu.Lock()
	delete(c.cache, scenario)
	c.mu.Unlock()
}

// decide applies the §F.3 contract:
//   - Any high-severity violation from routed_database_drivers disqualifies.
//   - Any high-severity violation from database_backoff also disqualifies
//     (a raw sql.Open defeats routing).
//   - Any medium-or-higher violation from routed_database_handle_capture
//     disqualifies.
func decide(summary *ViolationSummary) Eligibility {
	elig := Eligibility{Routed: true, Summary: summary}
	if summary == nil {
		return elig
	}

	var related []ViolationExcerpt
	for _, v := range summary.TopViolations {
		switch v.RuleID {
		case RuleRoutedDrivers:
			if severityAtLeast(v.Severity, "high") {
				elig.Routed = false
				related = append(related, v)
			}
		case RuleRoutedHandleCapture:
			if severityAtLeast(v.Severity, "medium") {
				elig.Routed = false
				related = append(related, v)
			}
		case RuleDatabaseBackoff:
			if severityAtLeast(v.Severity, "high") {
				elig.Routed = false
				related = append(related, v)
			}
		}
	}

	// If the scan didn't surface a top-violation excerpt for the routing
	// rules but the by-rule counts include them, treat as disqualifying
	// without specific file:line — surface as generic violations.
	if elig.Routed {
		for _, rc := range summary.ByRule {
			switch rc.RuleID {
			case RuleRoutedDrivers, RuleDatabaseBackoff:
				if rc.Count > 0 && severityAtLeast(rc.Severity, "high") {
					elig.Routed = false
					related = append(related, ViolationExcerpt{
						Severity: rc.Severity,
						RuleID:   rc.RuleID,
						Title:    rc.Title,
					})
				}
			case RuleRoutedHandleCapture:
				if rc.Count > 0 && severityAtLeast(rc.Severity, "medium") {
					elig.Routed = false
					related = append(related, ViolationExcerpt{
						Severity: rc.Severity,
						RuleID:   rc.RuleID,
						Title:    rc.Title,
					})
				}
			}
		}
	}

	elig.Violations = related
	return elig
}

func severityAtLeast(have, threshold string) bool {
	return severityWeight(have) >= severityWeight(threshold)
}

func severityWeight(sev string) int {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info", "informational":
		return 1
	default:
		return 0
	}
}
