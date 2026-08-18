package findingledger

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// Finding is the durable, run-independent identity and ranking input for one
// validation observation. Run IDs are deliberately absent from Identity.
type Finding struct {
	Asset      string    `json:"asset"`
	Version    string    `json:"version"`
	Check      string    `json:"check"`
	Viewport   string    `json:"viewport"`
	Theme      string    `json:"theme"`
	Kit        string    `json:"kit"`
	Severity   string    `json:"severity"`
	Adoptions  int       `json:"adoptions"`
	TargetRung int       `json:"targetRung"`
	FirstSeen  time.Time `json:"firstSeen"`
	LastSeen   time.Time `json:"lastSeen"`
	Message    string    `json:"message,omitempty"`
	Identity   string    `json:"identity"`
	RankReason string    `json:"rankReason"`
}

func Identity(asset, version, check, viewport, theme, kit string) string {
	value := strings.Join([]string{asset, version, check, viewport, theme, kit}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func Normalize(finding Finding) Finding {
	finding.Identity = Identity(finding.Asset, finding.Version, finding.Check, finding.Viewport, finding.Theme, finding.Kit)
	if finding.FirstSeen.IsZero() {
		finding.FirstSeen = finding.LastSeen
	}
	if finding.LastSeen.IsZero() {
		finding.LastSeen = finding.FirstSeen
	}
	finding.RankReason = rankReason(finding)
	return finding
}

func Merge(existing []Finding, observations []Finding) []Finding {
	byIdentity := make(map[string]Finding, len(existing)+len(observations))
	for _, finding := range append(existing, observations...) {
		finding = Normalize(finding)
		prior, ok := byIdentity[finding.Identity]
		if !ok {
			byIdentity[finding.Identity] = finding
			continue
		}
		if prior.FirstSeen.IsZero() || (!finding.FirstSeen.IsZero() && finding.FirstSeen.Before(prior.FirstSeen)) {
			prior.FirstSeen = finding.FirstSeen
		}
		if finding.LastSeen.After(prior.LastSeen) {
			prior.LastSeen = finding.LastSeen
		}
		prior.Message, prior.Severity, prior.Adoptions, prior.TargetRung = finding.Message, finding.Severity, finding.Adoptions, finding.TargetRung
		byIdentity[finding.Identity] = Normalize(prior)
	}
	out := make([]Finding, 0, len(byIdentity))
	for _, finding := range byIdentity {
		out = append(out, finding)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := score(out[i])
		right := score(out[j])
		if left != right {
			return left > right
		}
		return out[i].Identity < out[j].Identity
	})
	for i := range out {
		out[i].RankReason = rankReason(out[i])
	}
	return out
}

func score(finding Finding) int {
	return finding.Adoptions*100 + finding.TargetRung*10 + severityWeight(finding.Severity)
}

func severityWeight(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "error", "critical", "blocker":
		return 3
	case "warning", "warn":
		return 2
	default:
		return 1
	}
}

func rankReason(finding Finding) string {
	return "adoptions=" + itoa(finding.Adoptions) + " × 100 + targetRung=" + itoa(finding.TargetRung) + " × 10 + severity=" + itoa(severityWeight(finding.Severity))
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var out [20]byte
	index := len(out)
	for value > 0 {
		index--
		out[index] = digits[value%10]
		value /= 10
	}
	if negative {
		index--
		out[index] = '-'
	}
	return string(out[index:])
}
