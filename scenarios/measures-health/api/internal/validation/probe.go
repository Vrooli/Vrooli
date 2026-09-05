package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	measures "github.com/vrooli/measures-go"
)

// DefaultMeasuresMountPath is the conventional prefix under a scenario's API
// where the measures-go serve helper (Registry.Handler) is mounted. The probe
// appends measures.DefaultExecutePath to it. Adopters (Phase 5/6) mount the
// serve helper here; MEASURES_HEALTH_MEASURES_MOUNT_PATH overrides it.
const DefaultMeasuresMountPath = "/measures"

// HTTPProber behaviorally probes a declared measure against its owning
// scenario's live measures serve endpoint, reusing the measures-go HTTPExecutor.
// It is the production Prober: it resolves the scenario URL via api-core
// discovery (never client-computed), executes the measure, and checks the
// response conforms to the declared result shape.
//
// Safety: the probe NEVER executes a write/destructive measure (doing so would
// mutate the target). Such measures are reported as "not probed" — their
// declaration was already statically validated; their endpoint cannot be
// exercised without side effects.
type HTTPProber struct {
	executor  *measures.HTTPExecutor
	mountPath string
	now       func() time.Time
}

// NewHTTPProber constructs a prober whose resolver maps a scenario id to its
// measures base URL via api-core discovery + the conventional mount path.
func NewHTTPProber() *HTTPProber {
	mount := DefaultMeasuresMountPath
	p := &HTTPProber{mountPath: mount, now: time.Now}
	p.executor = measures.NewHTTPExecutor(measures.BaseURLResolverFunc(p.resolve))
	return p
}

// resolve maps a scenario id to the base URL of its measures serve endpoint.
func (p *HTTPProber) resolve(ctx context.Context, scenario string) (string, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + p.mountPath, nil
}

// Probe runs the measure end-to-end and checks the result conforms to the
// declared shape. skipped=true (with no failure) when the target is unreachable
// or the measure is intentionally not probed (write/destructive).
func (p *HTTPProber) Probe(ctx context.Context, scenario string, decl measures.MeasureDeclaration) (ok bool, detail string, skipped bool) {
	// Never exercise a mutating measure.
	if decl.Effect != measures.EffectRead || !decl.RunEligible {
		return true, fmt.Sprintf("not probed (effect=%s); declaration validated statically", decl.Effect), true
	}

	// The executor resolves by decl.Scenario; ensure it is set to the target.
	decl.Scenario = scenario

	// Resolve params deterministically from the first example question (no LLM):
	// the probe exercises endpoint liveness + result shape, not extraction
	// accuracy, so leftover needs[] are filled by declared defaults where present.
	params := p.probeParams(ctx, decl)

	result, err := p.executor.Execute(ctx, decl, params)
	if err != nil {
		// Unreachable / no endpoint mounted -> skip (degraded), not a false ERROR.
		return false, err.Error(), true
	}

	if conform, why := conformsToResult(decl, result); !conform {
		return false, why, false
	}
	return true, "answered conforming to result." + decl.Result.ValueField, false
}

// probeParams resolves deterministic params for the probe from the measure's
// first example question, falling back to declared defaults.
func (p *HTTPProber) probeParams(ctx context.Context, decl measures.MeasureDeclaration) map[string]string {
	question := ""
	if len(decl.Questions) > 0 {
		question = decl.Questions[0]
	}
	res, err := measures.ResolveParams(ctx, question, decl, measures.ResolveOptions{Now: p.now()})
	if err != nil || res.Params == nil {
		return map[string]string{}
	}
	return res.Params
}

// conformsToResult checks a probe response against the declared result shape:
// a scalar measure must return its value_field (a non-empty Value), a
// table/series must return Fields, and provenance must be stamped.
func conformsToResult(decl measures.MeasureDeclaration, r measures.MeasureResult) (bool, string) {
	if strings.TrimSpace(r.Provenance.ExecutedQuery) == "" {
		return false, "missing provenance.executed_query"
	}
	switch decl.Result.Kind {
	case measures.ResultScalar:
		if strings.TrimSpace(r.Value) == "" {
			return false, "scalar measure returned an empty value"
		}
	case measures.ResultTable, measures.ResultSeries:
		if len(r.Fields) == 0 {
			return false, string(decl.Result.Kind) + " measure returned no rows"
		}
	}
	return true, ""
}
