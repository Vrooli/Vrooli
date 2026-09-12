package memberflow

import "strings"

func ParseOperatingTopicCatalogStatus(raw string) OperatingTopicCatalogStatus {
	normalized := normalizeOperatingTopicCatalogStatus(raw)
	switch normalized {
	case "live":
		return OperatingTopicStatusLive
	case "live transitional":
		return OperatingTopicStatusLiveTransitional
	case "live system":
		return OperatingTopicStatusLiveSystem
	case "live but under consumed", "live under consumed":
		return OperatingTopicStatusLiveUnderConsumed
	case "target":
		return OperatingTopicStatusTarget
	case "old":
		return OperatingTopicStatusOld
	case "external":
		return OperatingTopicStatusExternal
	default:
		return OperatingTopicStatusUnknown
	}
}

func normalizeOperatingTopicCatalogStatus(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.ReplaceAll(raw, "-", " ")
	raw = strings.ReplaceAll(raw, "_", " ")
	return strings.Join(strings.Fields(raw), " ")
}

func operatingTopicCatalogStatusIsCurrent(status OperatingTopicCatalogStatus) bool {
	switch status {
	case OperatingTopicStatusLive, OperatingTopicStatusLiveTransitional, OperatingTopicStatusLiveSystem, OperatingTopicStatusLiveUnderConsumed:
		return true
	default:
		return false
	}
}

func expectedTopicCatalogQualifier(status OperatingTopicCatalogStatus) (string, bool) {
	switch status {
	case OperatingTopicStatusLive, OperatingTopicStatusLiveTransitional, OperatingTopicStatusLiveSystem, OperatingTopicStatusLiveUnderConsumed:
		return "", true
	case OperatingTopicStatusTarget:
		return string(OperatingGraphQualifierFuture), true
	case OperatingTopicStatusOld:
		return string(OperatingGraphQualifierOld), true
	case OperatingTopicStatusExternal:
		return string(OperatingGraphQualifierExternal), true
	default:
		return "", false
	}
}
