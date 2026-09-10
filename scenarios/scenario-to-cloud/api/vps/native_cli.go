package vps

import (
	"context"
	"path/filepath"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/release"
)

// The native control plane is never compiled at deploy time. The release
// build owns compilation (reproducible flags, recorded provenance) and binds
// the binary's sha256 and platform into the release manifest; delivery copies
// the release's bundle, manifest and binary through reach.Deliver, which
// proves the remote bytes match before the target owner verifies them again.

// runReleaseDeliver places the bundle and, for a built release, the release
// manifest and the native control plane for the negotiated platform.
func runReleaseDeliver(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	in := action.Inputs
	local := in["artifact_path"]
	if local == "" {
		local = e.bundlePath
	}
	delivery := reach.Delivery{Files: []reach.ArtifactFile{{Role: "bundle", LocalPath: local, RemotePath: in["destination"], SHA256: in["bundle_sha256"]}}}
	if in["release_id"] != "" {
		rel, err := release.LoadDir(filepath.Dir(local))
		if err != nil {
			return "", err
		}
		caps, err := e.reach.Negotiate(ctx, e.rt.Target)
		if err != nil && !reach.IsKind(err, reach.KindProtocolUnsupported) {
			return "", err
		}
		declared := rel.Manifest.NativeCLI.GOOS + "/" + rel.Manifest.NativeCLI.GOARCH
		if caps.Platform != "" && caps.Platform != declared {
			return "", apierrors.Newf(apierrors.CodeUnsupportedCapability, "release %s carries a %s control plane; the target is %s", rel.Digest, declared, caps.Platform).
				WithDetail("artifact", filepath.Base(rel.NativeCLIPath())).WithDetail("declared", declared).WithDetail("required", caps.Platform)
		}
		delivery.Files = append(delivery.Files,
			reach.ArtifactFile{Role: "release_manifest", LocalPath: rel.ManifestPath(), RemotePath: in["release_manifest"]},
			reach.ArtifactFile{Role: "native_cli", LocalPath: rel.NativeCLIPath(), RemotePath: in["native_cli"], Mode: 0o755, SHA256: rel.Manifest.NativeCLI.SHA256},
		)
	}
	receipt, err := e.reach.Deliver(ctx, e.rt.Target, delivery)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(receipt.Files))
	for _, f := range receipt.Files {
		parts = append(parts, f.Role+"="+f.SHA256[:minInt(12, len(f.SHA256))])
	}
	return "delivered " + strings.Join(parts, ","), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
