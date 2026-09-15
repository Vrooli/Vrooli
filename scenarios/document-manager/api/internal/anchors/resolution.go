package anchors

type Outcome string

const (
	Resolved         Outcome = "resolved"
	ResolvedDegraded Outcome = "resolved-degraded"
	Unresolved       Outcome = "unresolved"
	UnknownVersion   Outcome = "unknown-version"
	UnknownDocument  Outcome = "unknown-document"
	Forbidden        Outcome = "forbidden"
)

type Resolution struct {
	Outcome Outcome
	Region  string
}

// Resolve applies the six-outcome contract without reading regenerable parse
// output. Geometric and tabular coordinates are intrinsic to source bytes;
// logical coordinates require a same-version or recorded alignment.
func Resolve(uri URI, documentExists, versionExists, allowed bool, alignedRegion, stableRegion string) Resolution {
	if !documentExists {
		return Resolution{Outcome: UnknownDocument}
	}
	if !allowed {
		return Resolution{Outcome: Forbidden}
	}
	if !versionExists {
		return Resolution{Outcome: UnknownVersion}
	}
	if uri.Kind == KindGeometric || uri.Kind == KindTabular {
		return Resolution{Outcome: Resolved, Region: "source:" + string(uri.Kind)}
	}
	if alignedRegion != "" {
		return Resolution{Outcome: Resolved, Region: alignedRegion}
	}
	if stableRegion != "" && uri.Coordinates.StablePath != "" {
		return Resolution{Outcome: ResolvedDegraded, Region: stableRegion}
	}
	return Resolution{Outcome: Unresolved}
}
