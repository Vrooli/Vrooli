package gates

import "strings"

func containsCanonicalLayerMount(source string) bool {
	return strings.Contains(source, "BaseStyles") && strings.Contains(source, "<BaseStyles")
}
