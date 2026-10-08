// Package generality holds the phase 24 generality proofs: a scenario that
// did not exist when the cloud ramp was built is brought onto the ramp
// through declarations alone and driven through the whole ramp in-process
// (closure → executable plan → durable admission → worker execution against
// a fake `cloud-target` owner → health observation → certification evidence),
// once, in two environments on two targets, across an update that must keep
// its declared routes, and beside the generic fixtures whose plan digests
// must not move.
//
// The package is test-only by design: it imports the production packages and
// never modifies them, so a passing suite is evidence that no cloud source
// edit was needed for the newcomer fixture under fixtures/workloads.
package generality
