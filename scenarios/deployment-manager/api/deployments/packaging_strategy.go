package deployments

import "strings"

// resolvePackagingStrategy applies explicit deployment request precedence over
// the profile pin. The desktop ramp owns registry validation of the value.
func resolvePackagingStrategy(requestOverride string, profileSettings any) string {
	if value := strings.TrimSpace(requestOverride); value != "" {
		return value
	}
	settings, ok := profileSettings.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := settings["packaging_strategy"].(string)
	return strings.TrimSpace(value)
}
