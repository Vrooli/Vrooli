package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

func execFenceTestStore(path string, f managedUpdateExecFence) *managedUpdateExecStore {
	return &managedUpdateExecStore{path: path, resource: f.Resource, operation: f.OperationID, instance: f.Intent.InstanceID, now: time.Now, write: config.WriteOwnedFileDurable, lock: func(ctx context.Context) (func(), error) {
		return artifactlock.AcquireWithPathContext(ctx, "exec-fence-test:"+path, func(string) (string, error) { return path + ".lock", nil })
	}}
}

func execFenceTestInitial(t *testing.T) (managedUpdateExecFence, platform.ServiceRef) {
	t.Helper()
	u := updateTestInitial(t)
	p := ManagedUpdateProcess{Intent: updateTestIntent("candidate", u.Target)}
	updateTestObserveStarted(&p, u.Target, u.UpdatedAt)
	return managedUpdateExecFence{SchemaVersion: 1, Resource: u.Resource, OperationID: u.ID, Intent: *p.Intent, State: "prepared"}, *p.NativeRef
}

func TestManagedUpdateExecFenceAuthorizesOnlyOneConcurrentResourceExec(t *testing.T) {
	f, ref := execFenceTestInitial(t)
	s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
	if _, err := s.Prepare(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	var authorized atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Claim(context.Background(), f, ref)
			if err == nil {
				authorized.Add(1)
			} else if !errors.Is(err, ErrManagedUpdateExecClaimed) {
				t.Errorf("claim failed outside the once-only boundary: %v", err)
			}
		}()
	}
	wg.Wait()
	if authorized.Load() != 1 {
		t.Fatalf("%d callers received resource exec permission", authorized.Load())
	}
	if _, err := s.Prepare(context.Background(), f); !errors.Is(err, ErrManagedUpdateExecClaimed) {
		t.Fatalf("preparation reset a consumed exec permission: %v", err)
	}
	if _, err := s.Revoke(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(context.Background(), f, ref); !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatalf("late native dispatch executed after revocation: %v", err)
	}
	retired, err := s.read()
	if err != nil || retired.Claim == nil || retired.Claim.NativeInvocationID != ref.NativeInvocationID {
		t.Fatal("revocation forgot the native invocation that consumed exec permission")
	}
	if retired.RejectedClaim == nil || *retired.RejectedClaim != ref || retired.RejectedAt == nil {
		t.Fatal("revocation did not retain the exact rejected native delivery")
	}
}

func TestManagedUpdateExecFenceRejectedDeliverySurvivesLostAuditAcknowledgement(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprint(visible), func(t *testing.T) {
			f, ref := execFenceTestInitial(t)
			s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
			if _, err := s.Revoke(context.Background(), f); err != nil {
				t.Fatal(err)
			}
			s.write = func(path string, data []byte, mode os.FileMode) error {
				if visible {
					if err := config.WriteOwnedFileDurable(path, data, mode); err != nil {
						return err
					}
				}
				return errors.New("injected rejection audit acknowledgement loss")
			}
			if receipt, err := s.Claim(context.Background(), f, ref); !errors.Is(err, ErrManagedUpdateCommitUnknown) || receipt.Resource != "" {
				t.Fatal("unconfirmed rejection returned a resource exec receipt")
			}
			s = execFenceTestStore(s.path, f)
			if _, err := s.Claim(context.Background(), f, ref); !errors.Is(err, ErrManagedUpdateExecRevoked) {
				t.Fatal("audit failure restored resource exec permission")
			}
			after, err := s.read()
			if err != nil || after.State != "revoked" || after.RejectedClaim == nil || *after.RejectedClaim != ref {
				t.Fatal("exact rejected delivery did not survive reconciliation")
			}
		})
	}
}

func TestManagedUpdateExecFenceLostAcknowledgementNeverAuthorizesAnUnconfirmedExec(t *testing.T) {
	for _, afterRename := range []bool{false, true} {
		t.Run(fmt.Sprintf("after_rename_%t", afterRename), func(t *testing.T) {
			f, ref := execFenceTestInitial(t)
			s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
			if _, err := s.Prepare(context.Background(), f); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("injected exec claim acknowledgement loss")
			s.write = func(path string, data []byte, mode os.FileMode) error {
				if afterRename {
					if err := config.WriteOwnedFileDurable(path, data, mode); err != nil {
						return err
					}
				}
				return failure
			}
			if receipt, err := s.Claim(context.Background(), f, ref); !errors.Is(err, ErrManagedUpdateCommitUnknown) || receipt.Resource != "" {
				t.Fatalf("unconfirmed claim authorized resource exec: %+v %v", receipt, err)
			}
			s = execFenceTestStore(s.path, f)
			_, err := s.Claim(context.Background(), f, ref)
			if afterRename && !errors.Is(err, ErrManagedUpdateExecClaimed) || !afterRename && err != nil {
				t.Fatalf("reconstructed claim replay violated the once-only boundary: %v", err)
			}
			if _, err := s.Revoke(context.Background(), f); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Claim(context.Background(), f, ref); !errors.Is(err, ErrManagedUpdateExecRevoked) {
				t.Fatal("old resource exec permission was reusable after recovery")
			}
		})
	}
}

func TestManagedUpdateExecFenceMissingAndCorruptRecordsDoNotCreateExecPermission(t *testing.T) {
	f, ref := execFenceTestInitial(t)
	s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
	if _, err := s.Claim(context.Background(), f, ref); !os.IsNotExist(err) {
		t.Fatalf("missing fence created resource exec permission: %v", err)
	}
	if _, err := s.Revoke(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prepare(context.Background(), f); !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatal("missing-file recovery resurrected a revoked exec intent")
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := strings.Replace(string(data), `"state":"prepared"`, `"state":"claimed","state":"prepared"`, 1)
	if err := os.WriteFile(s.path, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(context.Background(), f, ref); err == nil {
		t.Fatal("duplicate state fields reset the resource exec fence")
	}
	if _, err := s.Revoke(context.Background(), f); err == nil {
		t.Fatal("unknown corruption was overwritten with an acknowledged fence")
	}
}

func TestManagedUpdateExecFenceClaimsAreExclusiveAcrossProcesses(t *testing.T) {
	if input := os.Getenv("VROOLI_EXEC_FENCE_TEST_INPUT"); input != "" {
		var fixture struct {
			Fence managedUpdateExecFence
			Ref   platform.ServiceRef
			Path  string
		}
		data, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		_, err = execFenceTestStore(fixture.Path, fixture.Fence).Claim(context.Background(), fixture.Fence, fixture.Ref)
		if err == nil {
			fmt.Println("EXEC_AUTHORIZED")
		} else if errors.Is(err, ErrManagedUpdateExecClaimed) {
			fmt.Println("EXEC_ALREADY_CONSUMED")
		} else {
			t.Fatal(err)
		}
		return
	}
	f, ref := execFenceTestInitial(t)
	root := t.TempDir()
	s := execFenceTestStore(filepath.Join(root, "fence.json"), f)
	if _, err := s.Prepare(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{"Fence": f, "Ref": ref, "Path": s.path})
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "input.json")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	results := make(chan string, 2)
	for range 2 {
		go func() {
			cmd := exec.Command(os.Args[0], "-test.run=^TestManagedUpdateExecFenceClaimsAreExclusiveAcrossProcesses$")
			cmd.Env = append(os.Environ(), "VROOLI_EXEC_FENCE_TEST_INPUT="+input)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("fixed exec-fence worker failed: %v %s", err, output)
			}
			results <- string(output)
		}()
	}
	authorized := 0
	for range 2 {
		if strings.Contains(<-results, "EXEC_AUTHORIZED") {
			authorized++
		}
	}
	if authorized != 1 {
		t.Fatalf("%d processes received permission to exec one resource intent", authorized)
	}
}

func TestManagedUpdateExecFenceRefusesAnotherOwnerPathAndNativeRoot(t *testing.T) {
	f, ref := execFenceTestInitial(t)
	s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
	foreign := f
	foreign.Resource = "another-resource"
	if _, err := s.Prepare(context.Background(), foreign); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("fence crossed its resource storage boundary: %v", err)
	}
	if _, err := s.Prepare(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	ref.RootPID = 0
	ref.RootStartTime = ""
	if _, err := s.Claim(context.Background(), f, ref); err == nil {
		t.Fatal("resource exec was authorized without a native root process")
	}
	loaded, err := s.read()
	if err != nil || loaded.State != "prepared" {
		t.Fatal("invalid native claim consumed the valid prepared fence")
	}
}

func TestManagedUpdateControlJSONHasOneBoundedInterpretation(t *testing.T) {
	for name, source := range map[string]string{
		"duplicate field":   `{"state":"claimed","state":"prepared"}`,
		"escaped duplicate": `{"state":"claimed","st\u0061te":"prepared"}`,
		"nested duplicate":  `{"record":{"native":{"id":1,"id":2}}}`,
		"trailing value":    `{} {}`,
		"unbounded depth":   strings.Repeat("[", 130) + "0" + strings.Repeat("]", 130),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateManagedUpdateJSON([]byte(source)); err == nil {
				t.Fatal("ambiguous or unbounded durable control document accepted")
			}
		})
	}
	if err := validateManagedUpdateJSON([]byte(`{"native":{"inode":18446744073709551615},"records":[null,true,"value"]}`)); err != nil {
		t.Fatalf("valid receipt lost exact numeric syntax: %v", err)
	}
}
