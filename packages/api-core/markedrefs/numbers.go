package markedrefs

// Number categories qualify the `num` marker. They state *why* an inline
// number in prose is intentional and owner-backed, so documentation tooling
// can suppress the drift-prone-count lint for it.
//
// A `num` marker carries exactly one category as its qualifier:
//
//	`num[target]:1000`     // a goal we are steering toward
//	`num[threshold]:100`   // a limit/gate baked into behavior
//	`num[price]:49`        // a monetary amount
//	`num[version]:1.25`    // a pinned version
//	`num[decision]:3`      // a count fixed by an explicit decision
//	`num[sot]:9`           // mirrors a source of truth (reserve-and-generate)
//
// These are deliberately *not* part of the global qualifier registry
// (KnownQualifiers): they are meaningful only for the `num` marker, and
// keeping them separate avoids implying that, say, `path[price]:...` is valid.
const (
	NumberCategoryTarget    = "target"
	NumberCategoryThreshold = "threshold"
	NumberCategoryPrice     = "price"
	NumberCategoryVersion   = "version"
	NumberCategoryDecision  = "decision"
	NumberCategorySoT       = "sot"
)

var numberCategorySpecs = []QualifierSpec{
	{NumberCategoryTarget, "A goal value the system is steering toward (changes by decision, not drift)."},
	{NumberCategoryThreshold, "A limit or gate baked into behavior (rate limit, retry count, timeout)."},
	{NumberCategoryPrice, "A monetary amount (SKU price, tier cost)."},
	{NumberCategoryVersion, "A pinned version, protocol number, or schema revision."},
	{NumberCategoryDecision, "A count fixed by an explicit operator decision."},
	{NumberCategorySoT, "Mirrors a named source of truth (reserve-and-generate candidate)."},
}

var knownNumberCategories = buildNumberCategorySet(numberCategorySpecs)

// KnownNumberCategories returns the supported `num` category registry in
// display order. The returned slice is a defensive copy.
func KnownNumberCategories() []QualifierSpec {
	out := make([]QualifierSpec, len(numberCategorySpecs))
	copy(out, numberCategorySpecs)
	return out
}

// IsKnownNumberCategory reports whether category is a recognized `num`
// justification category.
func IsKnownNumberCategory(category string) bool {
	return knownNumberCategories[category]
}

// NumberCategory returns the first recognized number category carried by ref
// and ok=true. For a non-`num` marker, or a `num` marker with no recognized
// category qualifier, it returns ("", false).
//
// A well-formed intentional number is `num[<category>]:<value>`; a `num`
// marker that returns ok=false is itself a lint finding ("intentional number,
// no stated reason").
func NumberCategory(ref Reference) (string, bool) {
	if ref.Marker != MarkerNum {
		return "", false
	}
	for _, q := range ref.Qualifiers {
		if knownNumberCategories[q] {
			return q, true
		}
	}
	return "", false
}

func buildNumberCategorySet(specs []QualifierSpec) map[string]bool {
	out := make(map[string]bool, len(specs))
	for _, spec := range specs {
		out[spec.Name] = true
	}
	return out
}
