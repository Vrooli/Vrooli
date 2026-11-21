package assetrung

import (
	"fmt"
)

// Rung is the dependency rank assigned to a catalog asset kind. Lower rungs
// are safe dependencies of higher rungs.
type Rung int

const (
	RungUnknown      Rung = -2
	RungFixture      Rung = -1
	RungFoundation   Rung = 0
	RungRuntime      Rung = 1
	RungPrimitive    Rung = 2
	RungComponent    Rung = 3
	RungComposition  Rung = 4
	RungPageTemplate Rung = 5
)

// UnknownKindError is returned when a catalog kind has no rung contract.
type UnknownKindError struct{ Kind string }

func (e UnknownKindError) Error() string { return fmt.Sprintf("unknown catalog asset kind %q", e.Kind) }

var kindRungs = map[string]Rung{
	"foundation":      RungFoundation,
	"runtime-hook":    RungRuntime,
	"runtime-service": RungRuntime,
	"adapter":         RungRuntime,
	"generator":       RungRuntime,
	"primitive":       RungPrimitive,
	"component":       RungComponent,
	"pattern":         RungComposition,
	"navigation":      RungComposition,
	"page-template":   RungPageTemplate,
	"fixture":         RungFixture,
}

// Of resolves a catalog kind and fails closed when the vocabulary is stale.
func Of(kind string) (Rung, error) {
	rung, ok := kindRungs[kind]
	if !ok {
		return RungUnknown, UnknownKindError{Kind: kind}
	}
	return rung, nil
}

func (r Rung) Name() string {
	switch r {
	case RungFixture:
		return "fixture"
	case RungFoundation:
		return "foundation"
	case RungRuntime:
		return "runtime"
	case RungPrimitive:
		return "primitive"
	case RungComponent:
		return "component"
	case RungComposition:
		return "composition"
	case RungPageTemplate:
		return "page-template"
	default:
		return "unknown"
	}
}

func (r Rung) String() string { return fmt.Sprintf("%d (%s)", int(r), r.Name()) }

func All() []Rung {
	return []Rung{RungFoundation, RungRuntime, RungPrimitive, RungComponent, RungComposition, RungPageTemplate}
}
