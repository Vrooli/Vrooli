package scenarioruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const dependencyConsumerPrefix = "scenario-dependency:"

// DependencyConsumerID is the durable owner identity for leases held by one
// running scenario instance. It is derived from the canonical instance slug
// so live and variant instances cannot release one another's dependencies.
func DependencyConsumerID(scenario, variant string) string {
	return dependencyConsumerPrefix + InstanceKey{Scenario: scenario, Variant: variant}.Slug()
}

// IsDependencyConsumerID reports whether an owner identity was created by the
// lifecycle dependency holder. Keeping the prefix private prevents callers
// from accidentally treating arbitrary consumer IDs as runtime instances.
func IsDependencyConsumerID(consumerID string) bool {
	return strings.HasPrefix(strings.TrimSpace(consumerID), dependencyConsumerPrefix)
}

// DependencyConsumerInstance resolves a lifecycle consumer identity back to
// its canonical scenario instance. It lets maintenance retain a dependency
// while that parent is still in its durable start-operation window, before the
// runtime instance row has been created.
func DependencyConsumerInstance(consumerID string) (InstanceKey, bool) {
	consumerID = strings.TrimSpace(consumerID)
	if !IsDependencyConsumerID(consumerID) {
		return InstanceKey{}, false
	}
	key, err := ParseInstanceKey(strings.TrimPrefix(consumerID, dependencyConsumerPrefix), "")
	if err != nil || key.Scenario == "" {
		return InstanceKey{}, false
	}
	return key, true
}

// DependencyLeaseID returns a bounded, retry-safe identity for one consumer's
// hold on one dependency instance.
func DependencyLeaseID(consumerID, scenario, variant string) string {
	key := InstanceKey{Scenario: scenario, Variant: variant}.Normalize()
	h := sha256.Sum256([]byte(strings.Join([]string{consumerID, key.Slug()}, "\x00")))
	return "dependency_" + hex.EncodeToString(h[:16])
}
