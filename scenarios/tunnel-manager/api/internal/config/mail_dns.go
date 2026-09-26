package config

import (
	"fmt"
	"strings"
)

// MergeSPFMechanism adds one provider mechanism to the single SPF record
// without discarding mechanisms owned by other systems. It is intentionally a
// pure operation; the caller decides whether the resulting TXT value may be
// applied to its DNS provider.
func MergeSPFMechanism(records []string, mechanism string) (string, error) {
	mechanism = strings.TrimSpace(mechanism)
	if mechanism == "" || !strings.Contains(mechanism, ":") {
		return "", fmt.Errorf("SPF mechanism must be a non-empty include or redirect value")
	}
	var spf string
	for _, record := range records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(record)), "v=spf1") {
			if spf != "" {
				return "", fmt.Errorf("multiple SPF records cannot be merged safely")
			}
			spf = strings.TrimSpace(record)
		}
	}
	if spf == "" {
		return "v=spf1 " + mechanism + " ~all", nil
	}
	fields := strings.Fields(spf)
	for _, field := range fields[1:] {
		if strings.EqualFold(strings.TrimPrefix(field, "+"), mechanism) {
			return spf, nil
		}
	}
	insertAt := len(fields)
	for i, field := range fields[1:] {
		if strings.HasSuffix(field, "all") {
			insertAt = i + 1
			break
		}
	}
	fields = append(fields, "")
	copy(fields[insertAt+1:], fields[insertAt:len(fields)-1])
	fields[insertAt] = mechanism
	return strings.Join(fields, " "), nil
}
