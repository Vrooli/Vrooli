// Package verify implements the navigation kind's reachability and
// deep-link policy verifier. It is consumed by navigation.Kind.Verify.
package verify

import (
	"encoding/json"
	"fmt"
	"sort"

	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/predicate"
)

// World is a concrete assignment of every declared context to a value.
type World map[string]string

// Key returns a deterministic string encoding so worlds can index maps.
func (w World) Key() string {
	keys := make([]string, 0, len(w))
	for k := range w {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var s string
	for _, k := range keys {
		s += k + "=" + w[k] + ";"
	}
	return s
}

// Clone returns a shallow copy.
func (w World) Clone() World {
	out := make(World, len(w))
	for k, v := range w {
		out[k] = v
	}
	return out
}

// EnumerateWorlds returns every valid context world. A context's
// valid_when constrains the partial world built so far: when false,
// that dimension is pinned to its declared default (treated as inert)
// instead of being enumerated. This keeps every world a complete
// assignment so downstream predicates always resolve.
func EnumerateWorlds(ctxs map[string]contract.Context) ([]World, error) {
	names := make([]string, 0, len(ctxs))
	for n := range ctxs {
		names = append(names, n)
	}
	sort.Strings(names)
	valuesByName := make(map[string][]string, len(names))
	defaults := make(map[string]string, len(names))
	preds := make(map[string]predicate.Predicate, len(names))
	for _, n := range names {
		ctx := ctxs[n]
		switch ctx.Kind {
		case "enum":
			valuesByName[n] = append([]string(nil), ctx.Values...)
		case "bool":
			valuesByName[n] = []string{"true", "false"}
		default:
			return nil, fmt.Errorf("context %q: unsupported kind %q", n, ctx.Kind)
		}
		d, err := decodeDefault(ctx)
		if err != nil {
			return nil, fmt.Errorf("context %q: %w", n, err)
		}
		defaults[n] = d
		p, err := predicate.Parse(ctx.ValidWhen)
		if err != nil {
			return nil, fmt.Errorf("context %q valid_when: %w", n, err)
		}
		preds[n] = p
	}

	worlds := []World{{}}
	for _, n := range names {
		var next []World
		for _, w := range worlds {
			active, err := preds[n].Eval(w.Lookup())
			if err != nil {
				// Predicate referenced a context not yet decided; treat
				// as inactive (pin default).
				active = false
			}
			if !active {
				nw := w.Clone()
				nw[n] = defaults[n]
				next = append(next, nw)
				continue
			}
			for _, v := range valuesByName[n] {
				nw := w.Clone()
				nw[n] = v
				next = append(next, nw)
			}
		}
		worlds = next
	}
	// Dedupe — pinning-to-default can produce identical worlds across
	// branches.
	seen := map[string]bool{}
	out := worlds[:0]
	for _, w := range worlds {
		k := w.Key()
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, w)
	}
	return out, nil
}

func decodeDefault(ctx contract.Context) (string, error) {
	if len(ctx.Default) == 0 {
		return "", fmt.Errorf("default required")
	}
	switch ctx.Kind {
	case "enum":
		var s string
		if err := json.Unmarshal(ctx.Default, &s); err != nil {
			return "", err
		}
		return s, nil
	case "bool":
		var b bool
		if err := json.Unmarshal(ctx.Default, &b); err != nil {
			return "", err
		}
		if b {
			return "true", nil
		}
		return "false", nil
	}
	return "", fmt.Errorf("unsupported kind %q", ctx.Kind)
}

// Lookup returns a Predicate lookup over this world.
func (w World) Lookup() predicate.Lookup {
	return func(name string) (string, bool) {
		v, ok := w[name]
		return v, ok
	}
}

// WorldsMatching returns the subset of worlds satisfying given.
func WorldsMatching(worlds []World, given string) ([]World, error) {
	p, err := predicate.Parse(given)
	if err != nil {
		return nil, fmt.Errorf("given %q: %w", given, err)
	}
	var out []World
	for _, w := range worlds {
		ok, err := p.Eval(w.Lookup())
		if err != nil {
			return nil, fmt.Errorf("given %q evaluation: %w", given, err)
		}
		if ok {
			out = append(out, w)
		}
	}
	return out, nil
}
