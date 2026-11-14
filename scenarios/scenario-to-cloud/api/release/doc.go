// Package release builds and verifies one immutable release artifact set.
//
// A release binds the deterministic mini-Vrooli bundle, the native
// control-plane binary for the target platform, the dependency closure digest
// and the nonsecret configuration digest to one release_digest (the canonical
// rule lives in packages/cloudrelease and is shared with the target-side
// verifier in internal/cloudtarget). The artifact tested is the artifact
// promoted: nothing is compiled or fetched at deploy time.
//
// Layout beneath the bundle store:
//
//	releases/<release_digest>/bundle.tar.gz
//	releases/<release_digest>/vrooli-<goos>-<goarch>
//	releases/<release_digest>/release-manifest.json
//	releases/<release_digest>/inputs.json
//	releases/<release_digest>/provenance/…        (signed releases only)
//	releases/<release_digest>/.complete           (written last)
//	releases/<release_digest>.lease.json          (artifact ownership)
//
// A directory without .complete is not a release: Store.Load refuses it and
// nothing downstream can activate it.
package release
