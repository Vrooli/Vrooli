//go:build linux

package resources

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

// This payload is launched as an owned private service by the exclusive update
// fixture below. It speaks only enough of the OpenRGB SDK handshake to exercise
// the resource's real loopback readiness checker; it has no device access.
func TestManagedUpdateExclusiveOpenRGBPayload(t *testing.T) {
	marker := os.Getenv("UPDATE_NATIVE_MARKER")
	if marker == "" {
		return
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", os.Getenv("RESOURCE_PORT_SDK")))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	version := uint32(6)
	if raw := os.Getenv("UPDATE_NATIVE_SDK_VERSION"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		version = uint32(parsed)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		serveManagedUpdateOpenRGBHandshake(conn, version)
	}
}

func serveManagedUpdateOpenRGBHandshake(conn net.Conn, version uint32) {
	defer conn.Close()
	var header [16]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil || string(header[:4]) != "ORGB" {
		return
	}
	bodySize := binary.LittleEndian.Uint32(header[12:16])
	if bodySize > 4096 {
		return
	}
	if _, err := io.CopyN(io.Discard, conn, int64(bodySize)); err != nil {
		return
	}
	var versionBody [4]byte
	binary.LittleEndian.PutUint32(versionBody[:], version)
	if writeManagedUpdateOpenRGBFrame(conn, 40, versionBody[:]) != nil || writeManagedUpdateOpenRGBFrame(conn, 51, []byte("private-fixture\x00")) != nil {
		return
	}
	if version >= 6 {
		var ackBody [8]byte
		binary.LittleEndian.PutUint32(ackBody[:4], 40)
		_ = writeManagedUpdateOpenRGBFrame(conn, 10, ackBody[:])
	}
}

func writeManagedUpdateOpenRGBFrame(w io.Writer, packetID uint32, body []byte) error {
	header := make([]byte, 16)
	copy(header[:4], "ORGB")
	binary.LittleEndian.PutUint32(header[8:12], packetID)
	binary.LittleEndian.PutUint32(header[12:16], uint32(len(body)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

func TestManagedUpdateCurrentHostExclusiveFullRunnerProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_EXCLUSIVE_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_EXCLUSIVE_PROBE=1 for private exclusive update and rollback fixtures")
	}
	for _, mode := range []string{"success", "target-failure"} {
		t.Run(mode, func(t *testing.T) {
			root, err := os.MkdirTemp("", "vrooli-openrgb-exclusive-update-")
			if err != nil {
				t.Fatal(err)
			}
			stateHome := isolateNativeUpdateFixtureStorage(t, root)
			t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
			defer cancel()

			resourceRoot := filepath.Join(projectRoot(t), "resources", "openrgb")
			m, err := manifestpkg.Load(filepath.Join(resourceRoot, "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			resource := "openrgb-exclusive-fixture"
			m.Name = resource
			m.ManagedService.UpdatePolicy.Strategy = "exclusive-stop-start"
			m.ManagedService.UpdatePolicy.CandidatePort = "sdk"
			m.Ports[0].Host = 0
			m.ManagedService.CredentialFiles = nil
			m.ManagedService.Environment = map[string]string{
				"UPDATE_NATIVE_MARKER":          filepath.Join(root, "old-executed"),
				"UPDATE_NATIVE_SDK_VERSION":     "6",
				"UPDATE_NATIVE_SDK_PORT_RENDER": "${RESOURCE_PORT_SDK}",
			}
			m.ManagedService.Arguments = []string{"-test.run=^TestManagedUpdateExclusiveOpenRGBPayload$", "-test.timeout=300s"}

			imageBytes, err := os.ReadFile(os.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			oldPath := filepath.Join(root, "old-image")
			if err := os.WriteFile(oldPath, append(imageBytes, 0), 0o755); err != nil {
				t.Fatal(err)
			}
			oldDigest, err := binaryfetch.FileDigestContext(ctx, oldPath)
			if err != nil {
				t.Fatal(err)
			}
			newDigest, err := binaryfetch.FileDigestContext(ctx, os.Args[0])
			if err != nil {
				t.Fatal(err)
			}

			reservation, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := reservation.Addr().(*net.TCPAddr).Port
			if err := reservation.Close(); err != nil {
				t.Fatal(err)
			}
			m.Ports[0].Host = port

			now := time.Now().UTC()
			m.ManagedService.Artifact.Version = "old-fixture"
			m.ManagedService.Artifact.SHA256ByPlatform = nil
			m.ManagedService.Artifact.SHA256 = oldDigest
			rollback, err := newManagedUpdateContractForResource(m, ManagedServiceArtifact{Resource: resource, Path: oldPath, Version: "old-fixture", SHA256: oldDigest}, resourceRoot)
			if err != nil {
				t.Fatal(err)
			}

			paths, err := resourceStoragePaths(resource)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
				t.Fatal(err)
			}
			supervisor, _, err := managedServiceSupervisorFor(resource)
			if err != nil {
				t.Fatal(err)
			}
			home := filepath.Join(root, "home")
			if err := os.MkdirAll(home, 0o700); err != nil {
				t.Fatal(err)
			}
			oldArgs := []string{"-test.run=^TestManagedUpdateExclusiveOpenRGBPayload$", "-test.timeout=300s"}
			oldEnv := []string{
				"HOME=" + home, "PATH=/usr/bin:/bin", "RESOURCE_PORT_SDK=" + strconv.Itoa(port),
				"RESOURCE_CONFIG_DIR=" + filepath.Join(paths.ConfigDir, "OpenRGB"),
				"UPDATE_NATIVE_MARKER=" + filepath.Join(root, "old-executed"),
				"UPDATE_NATIVE_SDK_VERSION=6", "UPDATE_NATIVE_SDK_PORT_RENDER=" + strconv.Itoa(port),
			}
			original, err := supervisor.StartContext(ctx, oldPath, m.ManagedService.Artifact, oldArgs, oldEnv, root, nil)
			if err != nil {
				t.Fatal(err)
			}
			var originalScope *platform.ServiceScopeRef
			var store *managedUpdateStore
			cleaned := false
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 45*time.Second)
				defer stop()
				if store != nil {
					current, lookupErr := store.Lookup("exclusive-" + strings.TrimPrefix(filepath.Base(root), "vrooli-openrgb-exclusive-update-"))
					if lookupErr == nil && current.Phase.Terminal() {
						if state, stateErr := supervisor.readState(); stateErr == nil && state.UpdateLaunch != nil {
							if stopErr := supervisor.Stop(cleanup); stopErr != nil {
								t.Errorf("serving fixture stop unconfirmed: %v", stopErr)
							}
						}
						for _, role := range []ManagedUpdateProcess{current.Primary, current.Restored} {
							if role.Intent != nil {
								if stopErr := cleanupManagedUpdateRole(cleanup, current, role); stopErr != nil {
									t.Errorf("fixture role cleanup unconfirmed: %v", stopErr)
								}
							}
						}
					} else if lookupErr == nil {
						t.Errorf("exclusive update remains unresolved at %s", store.path)
					}
				}
				if originalScope != nil {
					if _, stopErr := platform.StopServiceScope(cleanup, *originalScope); stopErr != nil {
						t.Errorf("original fixture cleanup unconfirmed: %v", stopErr)
					}
				} else {
					_ = supervisor.Stop(cleanup)
				}
				if cleaned && !t.Failed() {
					_ = os.RemoveAll(root)
				} else {
					t.Logf("exclusive fixture evidence retained at %s", root)
				}
			})

			readiness := m.HealthChecks[0]
			for {
				result, checkErr := resourcehealth.RunCheck(ctx, readiness, resourcehealth.Config{Env: []string{"RESOURCE_PORT_SDK=" + strconv.Itoa(port)}})
				if checkErr == nil && result.Healthy {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal("original OpenRGB SDK fixture did not become ready")
				case <-time.After(20 * time.Millisecond):
				}
			}
			scopePath, err := platform.ProcessScope(original.PID)
			if err != nil {
				t.Fatal(err)
			}
			_, stamp, _ := strings.Cut(original.Incarnation, ":")
			scope, err := platform.CaptureServiceScope(ctx, platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: resourceServiceScopeName(resource, "service"), Path: scopePath}, original.PID, stamp)
			if err != nil {
				t.Fatal(err)
			}
			originalScope = &scope

			m.ManagedService.Artifact.Version = "new-fixture"
			m.ManagedService.Artifact.SHA256 = newDigest
			m.ManagedService.Environment = map[string]string{
				"UPDATE_NATIVE_MARKER":          filepath.Join(root, "new-executed"),
				"UPDATE_NATIVE_SDK_VERSION":     "6",
				"UPDATE_NATIVE_SDK_PORT_RENDER": "${RESOURCE_PORT_SDK}",
			}
			if mode == "target-failure" {
				m.ManagedService.Environment["UPDATE_NATIVE_SDK_VERSION"] = "5"
			}
			target, err := newManagedUpdateContractForResource(m, ManagedServiceArtifact{Resource: resource, Path: os.Args[0], Version: "new-fixture", SHA256: newDigest}, resourceRoot)
			if err != nil {
				t.Fatal(err)
			}
			request := "exclusive-" + strings.TrimPrefix(filepath.Base(root), "vrooli-openrgb-exclusive-update-")
			operation := ManagedServiceUpdate{
				SchemaVersion: managedUpdateSchemaVersion, ID: request, RequestID: request, Resource: resource,
				Revision: 1, Phase: UpdatePrepared, CreatedAt: now, UpdatedAt: now,
				SourceRevision: updateTestDigest("openrgb-exclusive-fixture-source"), Target: target, Rollback: rollback,
				RollbackQualificationRequired: true, Original: original, OriginalScope: originalScope,
			}
			store, err = newManagedUpdateStore(resource)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := store.Create(ctx, operation, false); err != nil {
				t.Fatal(err)
			}

			goBinary, err := exec.LookPath("go")
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(ctx, goBinary, "run", "./cmd/vrooli", "managed-update-run", resource, request)
			cmd.Dir = projectRoot(t)
			cmd.Env = []string{
				"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_STATE_HOME=" + stateHome,
			}
			output, err := cmd.Output()
			if err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					t.Fatalf("exclusive update owner: %v: %s", err, exit.Stderr)
				}
				t.Fatal(err)
			}
			result, err := store.Lookup(request)
			if err != nil {
				t.Fatal(err)
			}
			if len(output) == 0 {
				t.Fatal("exclusive update owner returned no receipt")
			}
			wanted := UpdateSucceeded
			if mode == "target-failure" {
				wanted = UpdateFailedRestored
			}
			if result.Phase != wanted || result.Candidate.Intent != nil || result.RollbackProbe.Intent != nil || result.RollbackQualifiedAt == nil {
				t.Fatalf("exclusive update result=%s expected=%s; candidate=%t rollback_probe=%t", result.Phase, wanted, result.Candidate.Intent != nil, result.RollbackProbe.Intent != nil)
			}
			if mode == "target-failure" && (!result.PrimaryQuiescent || result.Restored.State == nil || result.Restored.Proof == nil) {
				t.Fatal("failed target did not reach a verified rollback service")
			}
			cleaned = true
		})
	}
}
