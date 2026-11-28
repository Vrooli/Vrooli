package markedrefs

// HasQualifier reports whether ref has qualifier.
func HasQualifier(ref Reference, qualifier string) bool {
	for _, q := range ref.Qualifiers {
		if q == qualifier {
			return true
		}
	}
	return false
}

// UnknownMarker reports whether ref's marker is not in the project registry.
func UnknownMarker(ref Reference) bool {
	return !IsKnownMarker(ref.Marker)
}

// UnknownQualifiers returns qualifiers that are not in the project registry.
func UnknownQualifiers(ref Reference) []string {
	var out []string
	for _, q := range ref.Qualifiers {
		if !IsKnownQualifier(q) {
			out = append(out, q)
		}
	}
	return out
}

// RequiresExistence reports whether validators should normally require the
// referenced value to exist in the current system.
//
// Domain validators remain authoritative. This helper encodes only the shared
// qualifier semantics from docs/reference/machine-readable-references.md.
func RequiresExistence(ref Reference) bool {
	if UnknownMarker(ref) {
		return false
	}
	if ref.Marker == MarkerLiteral {
		return false
	}
	for _, q := range []string{
		QualifierExample,
		QualifierOld,
		QualifierFuture,
		QualifierOptional,
		QualifierExternal,
		QualifierLiteral,
	} {
		if HasQualifier(ref, q) {
			return false
		}
	}
	return true
}

// IsLiteral reports whether ref is explicitly marked as a literal value.
func IsLiteral(ref Reference) bool {
	return ref.Marker == MarkerLiteral || HasQualifier(ref, QualifierLiteral)
}
