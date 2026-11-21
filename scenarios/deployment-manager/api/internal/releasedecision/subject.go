package releasedecision

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// SubjectKey is the Hub subject key of a release decision: one per profile,
// channel and target set, whatever the commit or artifact. A newer release of
// the same thing therefore updates or replaces the open decision instead of
// leaving the operator two, and the decision is closed when it is approved
// elsewhere or lapses. A part outside the Hub's key rule is cleaned, and the
// key is shortened behind a hash when too long, so two releases never share one.
func SubjectKey(profileID, channel string, targets []string) string {
	sorted := append([]string(nil), targets...)
	sort.Strings(sorted)
	whole := strings.Join([]string{profileID, channel, strings.Join(sorted, ",")}, "\x00")
	sum := sha256.Sum256([]byte(whole))
	key := "release:" + clean(profileID) + ":" + clean(channel) + ":" + hex.EncodeToString(sum[:6])
	if len(key) > 128 || !operatordecision.SubjectKeyRule.MatchString(key) {
		return "release:" + hex.EncodeToString(sum[:16])
	}
	return key
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return r
		}
		return '_'
	}, s)
}
