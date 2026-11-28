package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// MeasureRequest is the uniform request shape for executing any measure: the
// measure name plus its resolved string params. It is transport-agnostic — the
// Registry serves it over HTTP/JSON (Handler) and a scenario may also adapt it
// onto a Connect endpoint.
type MeasureRequest struct {
	Measure string            `json:"measure"`
	Params  map[string]string `json:"params,omitempty"`
}

// Provenance is the mandatory audit record stamped on every MeasureResult: the
// concrete query that produced the answer and when it was computed. Auto-answers
// are only trustworthy if they can be traced, so the serve helper guarantees
// ComputedAt is always set.
type Provenance struct {
	// ExecutedQuery describes the concrete query/aggregation that produced the
	// value (set by the compute func — e.g. the SQL or the resolved range).
	ExecutedQuery string `json:"executed_query"`
	// ComputedAt is when the value was computed (stamped by the helper).
	ComputedAt time.Time `json:"computed_at"`
}

// MeasureResult is the uniform response shape. For a scalar measure Value holds
// the answer; for table/series measures Fields holds the rows/points. Provenance
// is always populated.
type MeasureResult struct {
	Value      string              `json:"value,omitempty"`
	Fields     []map[string]string `json:"fields,omitempty"`
	Provenance Provenance          `json:"provenance"`
}

// ComputeFunc computes a measure's answer from its resolved params. It should
// set Provenance.ExecutedQuery; the Registry stamps ComputedAt if the func
// leaves it zero, so provenance is never empty.
type ComputeFunc func(ctx context.Context, req MeasureRequest) (MeasureResult, error)

type registered struct {
	decl MeasureDeclaration
	fn   ComputeFunc
}

// Registry holds a scenario's measure declarations + their compute funcs and
// executes them with uniform param validation and mandatory provenance. It is
// the serve helper of the contract library: a scenario builds one Registry,
// registers its measures, and mounts Handler() (or adapts Execute onto Connect).
type Registry struct {
	now      func() time.Time
	measures map[string]registered
}

// RegistryOption configures a Registry.
type RegistryOption func(*Registry)

// WithClock injects the time source used to stamp Provenance.ComputedAt (a seam
// for deterministic tests). Defaults to time.Now.
func WithClock(now func() time.Time) RegistryOption {
	return func(r *Registry) {
		if now != nil {
			r.now = now
		}
	}
}

// NewRegistry constructs an empty measure registry.
func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{
		now:      time.Now,
		measures: make(map[string]registered),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Register validates and registers a measure with its compute func. A duplicate
// name or an invalid declaration is an error.
func (r *Registry) Register(decl MeasureDeclaration, fn ComputeFunc) error {
	if fn == nil {
		return fmt.Errorf("measures: nil compute func for %q", decl.Name)
	}
	if err := decl.Validate(); err != nil {
		return err
	}
	if _, dup := r.measures[decl.Name]; dup {
		return fmt.Errorf("measures: duplicate measure %q", decl.Name)
	}
	r.measures[decl.Name] = registered{decl: decl, fn: fn}
	return nil
}

// Declarations returns the registered declarations in deterministic (sorted by
// name) order — what the central index (Phase 4) harvests.
func (r *Registry) Declarations() []MeasureDeclaration {
	names := make([]string, 0, len(r.measures))
	for n := range r.measures {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]MeasureDeclaration, 0, len(names))
	for _, n := range names {
		out = append(out, r.measures[n].decl)
	}
	return out
}

// Lookup returns a registered declaration by name.
func (r *Registry) Lookup(name string) (MeasureDeclaration, bool) {
	m, ok := r.measures[name]
	if !ok {
		return MeasureDeclaration{}, false
	}
	return m.decl, true
}

// Execute validates the request params against the measure declaration, runs
// the compute func, and guarantees Provenance.ComputedAt is stamped. An unknown
// measure, a missing required param, or an out-of-enum value is an error — the
// serve layer never executes an under-specified measure.
func (r *Registry) Execute(ctx context.Context, req MeasureRequest) (MeasureResult, error) {
	m, ok := r.measures[req.Measure]
	if !ok {
		return MeasureResult{}, fmt.Errorf("measures: unknown measure %q", req.Measure)
	}
	if err := validateParams(m.decl, req.Params); err != nil {
		return MeasureResult{}, err
	}
	out, err := m.fn(ctx, req)
	if err != nil {
		return MeasureResult{}, err
	}
	if out.Provenance.ComputedAt.IsZero() {
		out.Provenance.ComputedAt = r.now()
	}
	return out, nil
}

// validateParams enforces required-param presence and static-enum membership.
// Dynamic enums and free-form fields are not membership-checked here (their
// value space is runtime/unbounded); the resolver already gated those.
func validateParams(decl MeasureDeclaration, params map[string]string) error {
	var errs []string
	for _, name := range decl.ParamNames() {
		p := decl.Params[name]
		v, present := params[name]
		if !present || strings.TrimSpace(v) == "" {
			if p.Required {
				errs = append(errs, fmt.Sprintf("required param %q missing", name))
			}
			continue
		}
		if p.Type == ParamTypeTimeWindow {
			if !TimeWindowToken(v).Valid() {
				errs = append(errs, fmt.Sprintf("param %q value %q is not a canonical time-window token", name, v))
			}
			continue
		}
		if len(p.EnumValues) > 0 && !contains(p.EnumValues, v) {
			errs = append(errs, fmt.Sprintf("param %q value %q not in enum %v", name, v, p.EnumValues))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("measure %q param validation failed: %s", decl.Name, strings.Join(errs, "; "))
	}
	return nil
}

// Handler returns an http.Handler serving the registry over JSON. It exposes:
//
//	GET  <prefix>/declarations          → []MeasureDeclaration (the index harvest)
//	POST <prefix>/execute  {req}        → MeasureResult
//
// where <prefix> is the path the handler is mounted at. The handler is the
// framework-agnostic serve substrate; a Connect-based scenario can instead adapt
// Execute directly onto a generated service.
func (r *Registry) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/declarations", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, r.Declarations())
	})
	mux.HandleFunc("/execute", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var mr MeasureRequest
		if err := json.NewDecoder(req.Body).Decode(&mr); err != nil {
			http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
			return
		}
		out, err := r.Execute(req.Context(), mr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
