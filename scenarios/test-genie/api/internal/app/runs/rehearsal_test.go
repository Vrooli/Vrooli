package runs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"test-genie/internal/runmanager"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// TestCopiedRunEvidenceRehearsal is the copy-first Phase 10 proof path for
// historical Test Genie data. The source scenario is never opened by a
// production reader: its run index, run artifacts, and logs are copied into a
// temporary scenarios root before canonical wait/get and typed evidence APIs
// run twice. Normal suites skip it; operators opt in with a scenario path.
func TestCopiedRunEvidenceRehearsal(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0] [REQ:TESTGENIE-DESCRIPTOR-SNAPSHOT-P0] [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	source := strings.TrimSpace(os.Getenv("TEST_GENIE_RUN_REHEARSAL_SOURCE"))
	if source == "" {
		t.Skip("set TEST_GENIE_RUN_REHEARSAL_SOURCE to rehearse copied real run evidence")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(source) == sharedartifacts.CoverageRoot {
		source = filepath.Dir(source)
	}
	if _, err := os.Stat(sharedartifacts.RunsIndexPath(source)); err != nil {
		t.Fatalf("rehearsal source %s has no run index: %v", source, err)
	}

	root := t.TempDir()
	scenario := filepath.Base(source)
	copied := filepath.Join(root, scenario)
	for _, rel := range []string{
		filepath.Join(sharedartifacts.CoverageRoot, sharedartifacts.RunsIndexFile),
		sharedartifacts.RunsDir,
		sharedartifacts.LogsDir,
	} {
		if err := copyRehearsalPath(filepath.Join(source, rel), filepath.Join(copied, rel)); err != nil && !os.IsNotExist(err) {
			t.Fatalf("copy %s: %v", rel, err)
		}
	}
	// Index reads use an advisory sibling lock. Seed it before hashing so the
	// rehearsal distinguishes evidence mutation from the stable lock inode.
	if err := os.WriteFile(sharedartifacts.RunsIndexPath(copied)+".lock", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := rehearsalTreeDigest(filepath.Join(copied, sharedartifacts.CoverageRoot))
	if err != nil {
		t.Fatal(err)
	}

	index := sharedruns.NewIndex(copied)
	records, err := index.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatalf("no run records found under %s", source)
	}
	manager := runmanager.New(nil, root)
	defer manager.Shutdown()
	service := NewService(root, nil, nil, nil).SetArtifactRootResolver(testArtifactRoot(root))
	type counts struct {
		terminal, canonical, legacy, persistedCatalogs, discoveredCatalogs, artifacts int
	}
	var got counts
	for _, record := range records {
		if record.Status != sharedruns.StatusPassed && record.Status != sharedruns.StatusFailed && record.Status != sharedruns.StatusAborted {
			continue
		}
		got.terminal++
		firstWait, err := manager.Wait(context.Background(), scenario, record.RunID)
		if err != nil {
			t.Fatalf("%s wait: %v", record.RunID, err)
		}
		secondWait, err := manager.Wait(context.Background(), scenario, record.RunID)
		if err != nil {
			t.Fatalf("%s repeated wait: %v", record.RunID, err)
		}
		if !reflect.DeepEqual(firstWait, secondWait) {
			t.Fatalf("%s repeated wait projection changed", record.RunID)
		}
		if firstWait.Result == nil {
			got.legacy++
			if len(firstWait.DegradedReasons) == 0 {
				t.Fatalf("%s missing canonical snapshot was not degraded", record.RunID)
			}
		} else {
			got.canonical++
		}

		firstRun, err := service.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: scenario, RunId: record.RunID}))
		if err != nil {
			t.Fatalf("%s get run: %v", record.RunID, err)
		}
		secondRun, err := service.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: scenario, RunId: record.RunID}))
		if err != nil || !reflect.DeepEqual(firstRun.Msg, secondRun.Msg) {
			t.Fatalf("%s repeated run projection changed: %v", record.RunID, err)
		}

		_, catalogErr := sharedartifacts.ReadArtifactCatalog(copied, record.RunID)
		firstArtifacts, err := service.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Target: scenario, RunId: record.RunID}))
		if err != nil {
			t.Fatalf("%s list artifacts: %v", record.RunID, err)
		}
		secondArtifacts, err := service.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Target: scenario, RunId: record.RunID}))
		if err != nil || !reflect.DeepEqual(firstArtifacts.Msg, secondArtifacts.Msg) {
			t.Fatalf("%s repeated artifact projection changed: %v", record.RunID, err)
		}
		if catalogErr == nil {
			got.persistedCatalogs++
		} else if firstArtifacts.Msg.GetLegacyDiscovered() {
			got.discoveredCatalogs++
		} else {
			t.Fatalf("%s missing catalog was not labeled legacy: %v", record.RunID, catalogErr)
		}
		for _, artifact := range firstArtifacts.Msg.GetArtifacts() {
			got.artifacts++
			if _, err := service.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
				Target: scenario, RunId: record.RunID, ArtifactId: artifact.GetId(),
			})); err != nil {
				t.Fatalf("%s artifact %s failed integrity resolution: %v", record.RunID, artifact.GetId(), err)
			}
		}
	}
	after, err := rehearsalTreeDigest(filepath.Join(copied, sharedartifacts.CoverageRoot))
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("copied run evidence changed during read-only rehearsal: before=%s after=%s", before, after)
	}
	t.Logf("copied run rehearsal: terminal=%d canonical=%d legacy=%d persisted_catalogs=%d discovered_catalogs=%d artifacts=%d digest=%s", got.terminal, got.canonical, got.legacy, got.persistedCatalogs, got.discoveredCatalogs, got.artifacts, after)
}

func copyRehearsalPath(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return os.Symlink(target, destination)
	}
	if info.IsDir() {
		if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyRehearsalPath(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func rehearsalTreeDigest(root string) (string, error) {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(digest, "%s\x00", filepath.ToSlash(rel)); err != nil {
			return "", err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			_, _ = fmt.Fprintf(digest, "symlink:%s\x00", target)
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(digest, file); err != nil {
			_ = file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}
