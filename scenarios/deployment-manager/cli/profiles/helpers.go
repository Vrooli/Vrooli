package profiles

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func decodeProfile(data []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	normalizeProfile(&p)
	return p, nil
}

func normalizeProfile(p *Profile) {
	if p.Swaps == nil {
		p.Swaps = Swaps{}
	}
	if p.Secrets == nil {
		p.Secrets = map[string]interface{}{}
	}
	if p.Settings == nil {
		p.Settings = map[string]interface{}{}
	}
}

func ensureNestedMap(obj map[string]interface{}, key string) map[string]interface{} {
	if val, ok := obj[key]; ok {
		if cast, ok := val.(map[string]interface{}); ok && cast != nil {
			return cast
		}
	}
	newMap := map[string]interface{}{}
	obj[key] = newMap
	return newMap
}

func fetchVersionHistory(api *cliutil.APIClient, profileID string) (*ProfileHistory, error) {
	body, err := api.Get("/api/v1/profiles/"+profileID+"/versions", nil)
	if err != nil {
		return nil, err
	}
	var history ProfileHistory
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("parse version history: %w", err)
	}
	for i := range history.Versions {
		normalizeProfile(&history.Versions[i])
	}
	return &history, nil
}

func extractUpdatableProfileFields(source Profile) map[string]interface{} {
	return map[string]interface{}{
		"tiers":    source.Tiers,
		"swaps":    source.Swaps,
		"secrets":  source.Secrets,
		"settings": source.Settings,
	}
}

func versionNumber(v Profile) int {
	if v.Version != 0 {
		return v.Version
	}
	return 0
}

func findVersion(versions []Profile, version string) (Profile, bool) {
	for _, v := range versions {
		if fmt.Sprint(versionNumber(v)) == version {
			return v, true
		}
	}
	return Profile{}, false
}

func computeProfileDiff(previous, current Profile) map[string]map[string]interface{} {
	diff := map[string]map[string]interface{}{}

	if !intSlicesEqual(previous.Tiers, current.Tiers) {
		diff["tiers"] = map[string]interface{}{"from": previous.Tiers, "to": current.Tiers}
	}
	if !valuesEqual(previous.Swaps, current.Swaps) {
		diff["swaps"] = map[string]interface{}{"from": previous.Swaps, "to": current.Swaps}
	}
	if !valuesEqual(previous.Secrets, current.Secrets) {
		diff["secrets"] = map[string]interface{}{"from": previous.Secrets, "to": current.Secrets}
	}
	if !valuesEqual(previous.Settings, current.Settings) {
		diff["settings"] = map[string]interface{}{"from": previous.Settings, "to": current.Settings}
	}

	return diff
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func valuesEqual(a, b interface{}) bool {
	switch aVal := a.(type) {
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			if !valuesEqual(aVal[i], bVal[i]) {
				return false
			}
		}
		return true
	case []int:
		bVal, ok := b.([]int)
		if !ok {
			return false
		}
		return intSlicesEqual(aVal, bVal)
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if !valuesEqual(v, bVal[k]) {
				return false
			}
		}
		return true
	case map[string]string:
		bVal, ok := b.(map[string]string)
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if bVal[k] != v {
				return false
			}
		}
		return true
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

func intSliceToString(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

func profileShowReport(profile Profile) cliapp.ListReport {
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Profile: %s", firstNonEmpty(profile.ID, profile.Name)),
			fmt.Sprintf("Scenario: %s", profile.Scenario),
			fmt.Sprintf("Tiers: %s", fallbackProfileValue(intSliceToString(profile.Tiers), "(none)")),
		},
		ResultsHeading: "Details",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager profile versions %s", firstNonEmpty(profile.ID, profile.Name)),
			fmt.Sprintf("deployment-manager profile diff %s", firstNonEmpty(profile.ID, profile.Name)),
		},
	}
	report.Results = append(report.Results,
		fmt.Sprintf("Version: v%d", versionNumber(profile)),
		fmt.Sprintf("Swaps: %d", profile.Swaps.len()),
		fmt.Sprintf("Secrets keys: %d", len(profile.Secrets)),
		fmt.Sprintf("Settings keys: %d", len(profile.Settings)),
	)
	if len(profile.Swaps) > 0 {
		for _, swap := range profile.Swaps {
			report.Results = append(report.Results, fmt.Sprintf("Swap %s -> %s", swap.From, swap.To))
		}
	}
	return report
}

func fallbackProfileValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
