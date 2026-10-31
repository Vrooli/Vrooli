// Package blenderrenderer pins the reviewed Blender archive and verifies the
// complete managed tree before issuing a fixed, bounded version probe.
package blenderrenderer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
)

const (
	ToolName       = "blender-renderer"
	ProfileID      = "blender-5.2.2-linux-x64"
	BuildVersion   = "5.2.2 LTS"
	ArchiveRoot    = "blender-5.2.2-linux-x64"
	ArchiveURL     = "https://download.blender.org/release/Blender5.2/blender-5.2.2-linux-x64.tar.xz"
	ArchiveSHA256  = "84098912789dc450e95697c4184fb8a90acbe5111c2ba4aede3fecb57806a168"
	ArchiveSize    = int64(383295504)
	TreeSHA256     = "5fca420feab408276778511a84d7a2f5659bbc09aa5cb43b6d24ae59b220cb55"
	BinaryMember   = "blender-5.2.2-linux-x64/blender"
	BinarySHA256   = "bafec183e960819c76fe213c000cdb2284c53a87ef6791812f8728b51f07145b"
	BinarySize     = int64(174240960)
	LicenseMember  = "blender-5.2.2-linux-x64/license/license.md"
	LicenseSHA256  = "cdac130f03722dff2953d7da27035289bff8533d0bb3029d23e22ffe95a0bf7e"
	LicenseSize    = int64(185150)
	LicenseID      = "blender-gpl-3.0-or-later-with-component-notices"
	ProbeMaxOutput = 32 << 10
	probeTimeout   = 5 * time.Second
)

const licenseDetails = "Blender is GPL-3.0-or-later; bundled components retain their individual licenses. Redistribution has not been qualified."

type Artifact struct {
	ProfileID     string
	ArchiveURL    string
	ArchiveSHA256 string
	ArchiveSize   int64
	ArchiveRoot   string
	TreeSHA256    string
	BinaryMember  string
	BinarySHA256  string
	BinarySize    int64
	LicenseMember string
	LicenseSHA256 string
	LicenseSize   int64
	LicenseID     string
	LicenseDetail string
	Version       string
	Prerequisites []string
}

type State string

const (
	StateUnsupported State = "unsupported_platform"
	StateMissing     State = "missing"
	StateUnmanaged   State = "unmanaged"
	StatePartial     State = "partial"
	StateTampered    State = "tampered"
	StateProbeFailed State = "probe_failed"
	StatePresent     State = "artifact_present"
)

type Observation struct {
	State              State
	ArtifactIdentity   string
	EnvironmentID      string
	OS                 string
	Architecture       string
	ArchiveSHA256      string
	ArchiveSize        int64
	ExpectedTreeSHA256 string
	BinaryPath         string
	BinarySHA256       string
	BinarySize         int64
	Version            string
	LicenseID          string
	LicenseSHA256      string
	LicenseDetails     string
	Prerequisites      []string
	Detail             string
}

type Probe func(context.Context, string, ...string) ([]byte, error)

func PinnedArtifact() Artifact {
	return Artifact{
		ProfileID: ProfileID, ArchiveURL: ArchiveURL, ArchiveSHA256: ArchiveSHA256,
		ArchiveSize: ArchiveSize, ArchiveRoot: ArchiveRoot, TreeSHA256: TreeSHA256,
		BinaryMember: BinaryMember, BinarySHA256: BinarySHA256, BinarySize: BinarySize,
		LicenseMember: LicenseMember, LicenseSHA256: LicenseSHA256, LicenseSize: LicenseSize,
		LicenseID: LicenseID, LicenseDetail: licenseDetails, Version: BuildVersion,
		Prerequisites: []string{
			"Linux x86_64 and /lib64/ld-linux-x86-64.so.2; host glibc, libstdc++, libgcc, and X11-family libraries are required",
			"Cycles CPU/CUDA/OptiX availability and representative render behavior are unqualified",
		},
	}
}

func ArtifactIdentity(artifact Artifact) string {
	identity := struct {
		Schema        uint32 `json:"schema_version"`
		ProfileID     string `json:"profile_id"`
		ArchiveSHA256 string `json:"archive_sha256"`
		ArchiveSize   int64  `json:"archive_size_bytes"`
		ArchiveRoot   string `json:"archive_root"`
		TreeSHA256    string `json:"expected_tree_sha256"`
		BinaryMember  string `json:"binary_member"`
		BinarySHA256  string `json:"binary_sha256"`
		BinarySize    int64  `json:"binary_size_bytes"`
		LicenseMember string `json:"license_member"`
		LicenseSHA256 string `json:"license_sha256"`
		LicenseSize   int64  `json:"license_size_bytes"`
		LicenseID     string `json:"license_id"`
		Version       string `json:"version"`
	}{1, artifact.ProfileID, artifact.ArchiveSHA256, artifact.ArchiveSize, artifact.ArchiveRoot, artifact.TreeSHA256, artifact.BinaryMember, artifact.BinarySHA256, artifact.BinarySize, artifact.LicenseMember, artifact.LicenseSHA256, artifact.LicenseSize, artifact.LicenseID, artifact.Version}
	data, _ := json.Marshal(identity)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Resolve(ctx context.Context, optDir string, owned bool, artifact Artifact, goos, arch string, probe Probe) (Observation, error) {
	if ctx == nil {
		return Observation{}, errors.New("Blender profile context is nil")
	}
	response := Observation{
		State: StateMissing, ArtifactIdentity: ArtifactIdentity(artifact),
		EnvironmentID: goos + "-" + arch + "-host-runtime-prerequisites-unverified",
		OS:            goos, Architecture: arch, ArchiveSHA256: artifact.ArchiveSHA256,
		ArchiveSize: artifact.ArchiveSize, ExpectedTreeSHA256: artifact.TreeSHA256,
		BinaryPath:   filepath.Join(optDir, filepath.FromSlash(artifact.BinaryMember)),
		BinarySHA256: artifact.BinarySHA256, BinarySize: artifact.BinarySize,
		LicenseID: artifact.LicenseID, LicenseSHA256: artifact.LicenseSHA256,
		LicenseDetails: artifact.LicenseDetail, Prerequisites: append([]string(nil), artifact.Prerequisites...),
	}
	if goos != "linux" || arch != "amd64" {
		response.State = StateUnsupported
		response.Detail = "the inspected Blender archive supports Linux amd64 only"
		return response, nil
	}
	rootInfo, rootErr := os.Lstat(optDir)
	if errors.Is(rootErr, os.ErrNotExist) {
		if owned {
			response.State = StatePartial
			response.Detail = "the install ledger records the managed Blender directory, but it is missing"
		} else {
			response.Detail = "the managed Blender payload is not installed"
		}
		return response, nil
	}
	if rootErr != nil {
		response.State = StateTampered
		response.Detail = "the managed Blender directory could not be inspected"
		return response, nil
	}
	if !owned {
		response.State = StateUnmanaged
		response.Detail = "a Blender payload exists outside the recorded managed-tool directory; refusing to inspect or execute it"
		return response, nil
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		response.State = StateTampered
		response.Detail = "the managed Blender root is not a regular directory"
		return response, nil
	}
	binaryInfo, err := os.Lstat(response.BinaryPath)
	if errors.Is(err, os.ErrNotExist) {
		response.State = StatePartial
		response.Detail = "the pinned Blender executable is missing from its managed tree"
		return response, nil
	}
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Mode()&os.ModeSymlink != 0 || binaryInfo.Size() != artifact.BinarySize || binaryInfo.Mode().Perm()&0o111 == 0 {
		response.State = StateTampered
		response.Detail = "the pinned Blender executable has unexpected type, mode, or size"
		return response, nil
	}
	treeDigest, err := binaryfetch.TreeDigestContext(ctx, optDir)
	if err != nil {
		if ctx.Err() != nil {
			return Observation{}, ctx.Err()
		}
		response.State = StateTampered
		response.Detail = "the managed Blender tree contains an unsafe or unreadable entry"
		return response, nil
	}
	if treeDigest != artifact.TreeSHA256 {
		response.State = StateTampered
		response.Detail = "the managed Blender tree differs from the pinned archive content"
		return response, nil
	}
	if probe == nil {
		probe = VersionProbe
	}
	output, err := probe(ctx, response.BinaryPath, "--version")
	if err != nil {
		response.State = StateProbeFailed
		response.Detail = "the fixed Blender version probe failed or exceeded its bound; host runtime prerequisites remain unverified"
		return response, nil
	}
	line := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
	if line != "Blender "+artifact.Version {
		response.State = StateProbeFailed
		response.Detail = "the pinned Blender tree did not report the expected version"
		return response, nil
	}
	response.State = StatePresent
	response.Version = line
	response.Detail = "managed tree digest, pinned executable, and bounded version probe match; OS prerequisites and renderer workflow remain unqualified"
	return response, nil
}

func VersionProbe(ctx context.Context, path string, args ...string) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("Blender version probe context is nil")
	}
	if len(args) != 1 || args[0] != "--version" {
		return nil, errors.New("only the fixed Blender --version probe is supported")
	}
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	command := exec.CommandContext(probeCtx, path, "--version")
	command.Env = []string{"LANG=C", "LC_ALL=C"}
	var output boundedOutput
	output.limit = ProbeMaxOutput
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return nil, err
	}
	if output.overflow {
		return nil, errors.New("Blender version output exceeded its bound")
	}
	return bytes.Clone(output.data.Bytes()), nil
}

type boundedOutput struct {
	data     bytes.Buffer
	limit    int
	overflow bool
}

func (output *boundedOutput) Write(value []byte) (int, error) {
	if len(value) > output.limit-output.data.Len() {
		output.overflow = true
		return 0, errors.New("Blender version output exceeded its bound")
	}
	return output.data.Write(value)
}

func RuntimePlatform() (string, string) { return runtime.GOOS, runtime.GOARCH }
