// Package cloudtarget is the target-local owner of cloud deployment state.
//
// It runs inside the native vrooli binary on the deployment target and is
// exposed as `vrooli cloud-target <verb>`. Every effectful verb is scoped to
// one (deployment, operation, step, fence) tuple, writes a receipt beneath the
// runtime home, refuses a stale fence, and replays its receipt on a re-run.
// There is no generic shell verb: host repairs are delegated to the privilege
// broker's fixed action vocabulary and process control to the lifecycle owner.
//
// Layout beneath <runtime-home>/cloud/deployments/<deployment-id>/:
//
//	releases/<release-digest>/                 complete, immutable release tree
//	releases/<release-digest>.staging-<op>/    in-flight stage (never activatable)
//	active-release.json                        durable active/previous pointer
//	activation-intent.json                     present only during an activation
//	operations/<operation-id>/<step>.json      one receipt per (operation, step)
//	fence.json                                 highest fence accepted so far
package cloudtarget
