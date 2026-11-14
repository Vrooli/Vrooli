package main

import (
	"encoding/json"
	"fmt"
	"io"

	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/vroolicli"
)

func rootVersionInfo() vroolicli.VersionInfo {
	digest, _ := buildinfo.RuntimeHomeSnapshotDigest()
	return vroolicli.VersionInfo{CLIVersion: cliVersion, PlatformVersion: vrooliVersion, BuildIdentity: buildinfo.Fingerprint, RuntimeHomeContractSHA256: digest, PersistentStateContractSHA256: runtimeapp.NativePersistentStateContract()}
}

// Basic version probes must not repair PATH shims, register sessions, open
// command logs or initialize host state before an artifact is admitted.
func writeStandaloneVersion(args []string, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 2 && (args[0] == "version" || args[0] == "--version") && args[1] == "--state-manifest" {
		return writeStateManifest(stdout, stderr)
	}
	query, jsonOut := false, false
	if len(args) == 1 {
		query = args[0] == "version" || args[0] == "--version"
	}
	if len(args) == 2 {
		query = (args[0] == "version" || args[0] == "--version") && args[1] == "--json" || args[0] == "--json" && (args[1] == "version" || args[1] == "--version")
		jsonOut = query
	}
	if !query {
		return false, 0
	}
	root, _ := resolveSourceRootFn()
	if err := vroolicli.WriteVersion(stdout, root, rootcli.GlobalOptions{JSON: jsonOut}, rootVersionInfo()); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	return true, 0
}

// writeStateManifest prints this executable's per-owner persistent-state
// contract as canonical JSON. Activation admission reads it from both images
// to name the owner and stored shape behind a contract difference.
func writeStateManifest(stdout, stderr io.Writer) (bool, int) {
	manifest, err := runtimeapp.NativePersistentStateManifest()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "persistent-state contract is unqualified:", err)
		return true, 1
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	if _, err := stdout.Write(append(data, '\n')); err != nil {
		return true, 1
	}
	return true, 0
}
