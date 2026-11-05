package vps

import (
	"strings"
	"testing"

	"scenario-to-cloud/execplan"
)

// [REQ:STC-P0-028] The shell preview is derived from the typed invocations:
// it renders the same verbs and arguments the executor dispatches, through
// the SSH adapter's quoting, and marks the actions that carry no target verb
// (credential values, data disposition) as descriptions rather than
// commands. Nothing in it is executed.
func TestShellPreviewIsDerivedFromArgv(t *testing.T) {
	cc := edgeContext("production", "")
	remote := func(argv []string) string { return "ssh host " + strings.Join(argv, " ") }
	scp := func(local, remotePath string) string { return "scp " + local + " host:" + remotePath }
	stage := execplan.Action{ID: execplan.OpReleaseStage, OwnerOperation: execplan.OpReleaseStage, Inputs: map[string]string{"release_id": strings.Repeat("a", 64), "release_manifest": "/root/Vrooli/.vrooli/cloud/bundles/x/release-manifest.json", "archive": "/root/Vrooli/.vrooli/cloud/bundles/x/bundle.tar.gz"}}
	preview := ShellPreviewFor(stage, cc, remote, scp)
	for _, want := range []string{"cloud-target release stage", "--operation op-1", "--fence 1", "--release-manifest /root/Vrooli/.vrooli/cloud/bundles/x/release-manifest.json"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview %q lacks %q", preview, want)
		}
	}
	deliver := execplan.Action{ID: execplan.OpReleaseDeliver, OwnerOperation: execplan.OpReleaseDeliver, Inputs: map[string]string{"artifact_path": "/tmp/bundle.tar.gz", "destination": "/root/Vrooli/.vrooli/cloud/bundles/x/bundle.tar.gz", "release_id": strings.Repeat("a", 64), "release_manifest": "/root/Vrooli/.vrooli/cloud/bundles/x/release-manifest.json", "native_cli": "/root/Vrooli/.vrooli/bin/vrooli"}}
	preview = ShellPreviewFor(deliver, cc, remote, scp)
	if !strings.HasPrefix(preview, "scp /tmp/bundle.tar.gz host:") || !strings.Contains(preview, "host:/root/Vrooli/.vrooli/bin/vrooli") {
		t.Fatalf("deliver preview = %q", preview)
	}
	creds := execplan.Action{ID: execplan.OpCredentialsProvision, OwnerOperation: execplan.OpCredentialsProvision, Inputs: map[string]string{"descriptors": "db-password:per_install_generated"}}
	preview = ShellPreviewFor(creds, cc, remote, scp)
	if !strings.HasPrefix(preview, "(") || strings.Contains(preview, "ssh ") {
		t.Fatalf("credential provisioning must render as a description, not a command: %q", preview)
	}
	bare := execplan.Action{ID: execplan.OpReleaseStage, OwnerOperation: execplan.OpReleaseStage, Inputs: map[string]string{"archive": "/x"}}
	if preview := ShellPreviewFor(bare, cc, remote, scp); !strings.Contains(preview, "release_manifest_missing") && !strings.Contains(preview, "not part of a built release") {
		t.Fatalf("a bare bundle must render its refusal: %q", preview)
	}
}
