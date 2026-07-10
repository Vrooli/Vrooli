package agentpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// PermissionProjection is the resource-local managed subset expressed in the
// common three-bucket form. It is intentionally not a native config model.
type PermissionProjection struct {
	Allow []string
	Ask   []string
	Deny  []string
}

type PermissionPlanResult struct {
	SchemaVersion      string             `json:"schema_version"`
	Runner             string             `json:"runner"`
	Scope              string             `json:"scope,omitempty"`
	DesiredDigest      string             `json:"desired_digest"`
	DesiredFingerprint string             `json:"desired_fingerprint"`
	LiveFingerprint    string             `json:"live_fingerprint"`
	Drift              bool               `json:"drift"`
	Changes            []string           `json:"changes"`
	NativePaths        []string           `json:"native_paths"`
	Enforcement        EnforcementPosture `json:"enforcement"`
}

func PlanPermissionProjection(runner string, document PermissionDocument, documentBytes []byte, live PermissionProjection, nativePaths []string, posture EnforcementPosture) PermissionPlanResult {
	allow, ask, deny := PermissionPatterns(document)
	desired := PermissionProjection{Allow: allow, Ask: ask, Deny: deny}
	liveFingerprint := PermissionProjectionFingerprint(live)
	desiredFingerprint := PermissionProjectionFingerprint(desired)
	changes := projectionChanges(live, desired)
	paths := append([]string(nil), nativePaths...)
	sort.Strings(paths)
	return PermissionPlanResult{
		SchemaVersion:      PermissionDocumentSchemaVersion,
		Runner:             runner,
		Scope:              document.Scope,
		DesiredDigest:      PermissionDocumentDigest(documentBytes),
		DesiredFingerprint: desiredFingerprint,
		LiveFingerprint:    liveFingerprint,
		Drift:              desiredFingerprint != liveFingerprint,
		Changes:            changes,
		NativePaths:        paths,
		Enforcement:        posture,
	}
}

func PermissionProjectionFingerprint(projection PermissionProjection) string {
	canonical := struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
	}{Allow: sortedProjection(projection.Allow), Ask: sortedProjection(projection.Ask), Deny: sortedProjection(projection.Deny)}
	data, _ := json.Marshal(canonical)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func projectionChanges(live, desired PermissionProjection) []string {
	changes := make([]string, 0)
	for _, bucket := range []struct {
		name string
		live []string
		want []string
	}{{"allow", live.Allow, desired.Allow}, {"ask", live.Ask, desired.Ask}, {"deny", live.Deny, desired.Deny}} {
		if equalStrings(bucket.live, bucket.want) {
			continue
		}
		changes = append(changes, fmt.Sprintf("replace %s rules", bucket.name))
	}
	return changes
}

func equalStrings(left, right []string) bool {
	left = sortedProjection(left)
	right = sortedProjection(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedProjection(values []string) []string {
	copy := append([]string(nil), values...)
	sort.Strings(copy)
	return copy
}
