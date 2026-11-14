// Package routing defines the shared provider-profile eligibility and ordering
// contract used by AI Gateway's generic and typed inference paths.
package routing

import "strings"

// Admits reports whether a candidate locality satisfies the request's privacy
// class and profile. Empty/unspecified profile and privacy preserve the
// gateway's established local-first/default behavior.
func Admits(profile, privacyClass, locality string) (bool, string) {
	profile = normalize(profile, "PROFILE_")
	privacyClass = normalize(privacyClass, "PRIVACY_CLASS_")
	locality = strings.ToLower(strings.TrimSpace(locality))
	if locality != "local" && locality != "remote" {
		return false, "candidate locality is unknown"
	}
	if privacyClass == "SECRET" && locality == "remote" {
		return false, "secret requests cannot route to remote providers"
	}
	if locality == "remote" && privacyClass != "PUBLIC" && privacyClass != "INTERNAL" && privacyClass != "CONFIDENTIAL" && privacyClass != "SECRET" {
		return false, "remote providers require a recognized explicit privacy classification"
	}
	switch profile {
	case "", "UNSPECIFIED", "LOCAL_FIRST", "QUALITY_FIRST", "CHEAP_FIRST":
		return true, ""
	case "LOCAL_ONLY":
		if locality != "local" {
			return false, "local-only profile forbids remote providers"
		}
	case "REMOTE_ONLY":
		if locality != "remote" {
			return false, "remote-only profile forbids local providers"
		}
	case "PRIVACY_SENSITIVE":
		if (privacyClass == "CONFIDENTIAL" || privacyClass == "SECRET") && locality != "local" {
			return false, "privacy-sensitive confidential requests require local providers"
		}
	default:
		return false, "request profile is unknown"
	}
	return true, ""
}

// Rank returns a stable locality/provider rank for profiles that order
// eligible candidates. It intentionally implements the gateway's declared
// locality heuristic; QUALITY_FIRST and CHEAP_FIRST are not measured quality
// or cost optimization.
func Rank(profile, locality, provider string) int {
	profile = normalize(profile, "PROFILE_")
	locality = strings.ToLower(strings.TrimSpace(locality))
	base := 10
	switch profile {
	case "REMOTE_ONLY":
		if locality == "remote" {
			base = 0
		}
	case "QUALITY_FIRST":
		if locality == "remote" {
			base = 0
		} else {
			base = 5
		}
	case "LOCAL_ONLY", "LOCAL_FIRST", "CHEAP_FIRST", "PRIVACY_SENSITIVE":
		if locality == "local" {
			base = 0
		} else {
			base = 5
		}
	case "", "UNSPECIFIED":
		// Preserve the established unspecified-profile order: declared catalog
		// order dominates except for the existing Ollama tie-break.
		base = 10
	default:
		return 100
	}
	if strings.EqualFold(strings.TrimSpace(provider), "ollama") {
		return base
	}
	return base + 1
}

func normalize(value, prefix string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.TrimPrefix(value, prefix)
}
