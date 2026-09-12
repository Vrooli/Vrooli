package zones

// Evidence is one typed justification behind a converged zone classification.
// It folds directly into an attestation Citation at the handler boundary.
type Evidence struct {
	Kind    string // "manifest" | "architecture_doc" | "import_graph"
	Detail  string
	Locator string
}

// Converged is the three-signal zone verdict for one package: the DERIVED zone
// (template manifest path patterns), the DECLARED zone (ARCHITECTURE.md Zone
// Map), and the import-graph reality (import-direction consistency). The
// resolved Zone is always the code-derived one — a declared/derived
// disagreement is reported as drift, never used to override the code (D3).
type Converged struct {
	Zone          string // resolved zone = derived (code reality)
	DerivedZone   string
	DeclaredZone  string // canonical zone from ARCHITECTURE.md, "" when absent/unmapped
	DeclaredLayer string // raw label as written
	Confidence    float64
	Evidence      []Evidence
	Drift         bool // declared present and disagrees with derived
}

// Converge fuses the three zone signals for a package. importConsistent is nil
// when the import-graph signal was not evaluated; true/false otherwise.
func Converge(repoPath string, derived Info, dzm DeclaredZoneMap, importConsistent *bool) Converged {
	out := Converged{Zone: derived.Zone, DerivedZone: derived.Zone}

	// DERIVED signal (manifest path patterns).
	confidence := 0.3
	if derived.Zone != Unknown {
		confidence = 0.6
		out.Evidence = append(out.Evidence, Evidence{
			Kind:    "manifest",
			Detail:  "classified as " + derived.Zone + " by template manifest path patterns",
			Locator: repoPath,
		})
	} else {
		out.Evidence = append(out.Evidence, Evidence{
			Kind:    "manifest",
			Detail:  "no template manifest path pattern matched",
			Locator: repoPath,
		})
	}

	// DECLARED signal (ARCHITECTURE.md Zone Map).
	if dz, layer, ok := dzm.ZoneFor(repoPath); ok {
		out.DeclaredZone = dz
		out.DeclaredLayer = layer
		out.Evidence = append(out.Evidence, Evidence{
			Kind:    "architecture_doc",
			Detail:  "ARCHITECTURE.md declares layer " + layer,
			Locator: ArchitectureDocPath,
		})
		switch {
		case derived.Zone != Unknown && sameZoneFamily(dz, derived.Zone):
			confidence += 0.3 // declared agrees with code (same responsibility family)
		case dz != "" && derived.Zone != Unknown && !sameZoneFamily(dz, derived.Zone):
			out.Drift = true
			confidence = 0.4 // contradicted: doc disagrees with code
		}
	}

	// REALITY signal (import-graph direction consistency).
	if importConsistent != nil {
		if *importConsistent {
			confidence += 0.1
			out.Evidence = append(out.Evidence, Evidence{
				Kind:    "import_graph",
				Detail:  "import directions are consistent with the " + derived.Zone + " zone",
				Locator: repoPath,
			})
		} else {
			confidence -= 0.1
			out.Evidence = append(out.Evidence, Evidence{
				Kind:    "import_graph",
				Detail:  "imports reach a higher zone than " + derived.Zone + " should depend on",
				Locator: repoPath,
			})
		}
	}

	out.Confidence = clamp01(confidence)
	return out
}

// sameZoneFamily reports whether two canonical zones share a responsibility
// family. domain / substrate / composition-root all live under api/internal and
// form the "core" family — the ARCHITECTURE.md Zone Map declares api/internal as
// core but cannot lexically distinguish a domain folder from a substrate folder
// (that finer split is the derived signal's job, resolved via domain
// membership). Drift is therefore only reported on a genuine cross-family
// disagreement (e.g. a doc-declared transport package the code derives as core).
func sameZoneFamily(a, b string) bool {
	return zoneFamily(a) == zoneFamily(b)
}

func zoneFamily(zone string) string {
	switch zone {
	case Domain, Substrate, CompositionRoot:
		return "core"
	default:
		return zone
	}
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
