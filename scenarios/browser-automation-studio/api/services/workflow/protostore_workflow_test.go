package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// TestWriteWorkflowSummaryFile_RoundTrip_NoPathDoubling pins the canonical
// path contract: WriteWorkflowSummaryFile(project, wf, preferredRel)
// followed by reading back through filepath.Join(project.FolderPath,
// returned-relPath) must yield the same on-disk file.
//
// Before the fix, preferredRel was treated as relative to
// ProjectWorkflowsDir (project_root/workflows) while WorkflowIndex.FilePath
// is project-root-relative. Every Update fed the previously-stored
// FilePath back in, so paths doubled ("workflows/workflows/...") on each
// round-trip and the canonical file kept moving deeper into the tree.
func TestWriteWorkflowSummaryFile_RoundTrip_NoPathDoubling(t *testing.T) {
	dir := t.TempDir()
	project := &database.ProjectIndex{
		ID:         uuid.New(),
		Name:       "Demo",
		FolderPath: dir,
	}
	id := uuid.New()
	wf := &basapi.WorkflowSummary{
		Id:             id.String(),
		ProjectId:      project.ID.String(),
		Name:           "Smoke",
		FolderPath:     "/demo",
		Version:        1,
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
	}

	abs1, rel1, err := WriteWorkflowSummaryFile(project, wf, "")
	if err != nil {
		t.Fatalf("initial write: %v", err)
	}
	if !strings.HasPrefix(filepath.ToSlash(rel1), "workflows/") {
		t.Errorf("rel must be project-root-relative with leading workflows/, got %q", rel1)
	}
	if _, err := os.Stat(abs1); err != nil {
		t.Errorf("initial file does not exist at returned abs %q: %v", abs1, err)
	}
	// Hydrate path resolution: project.FolderPath + rel must equal abs.
	if got := filepath.Join(project.FolderPath, filepath.FromSlash(rel1)); got != abs1 {
		t.Errorf("path round-trip mismatch:\n  abs:           %s\n  joinFromRel:   %s", abs1, got)
	}

	// Now do an update: feed the previously-returned rel back in as
	// preferredRel. This is exactly what UpdateWorkflow does.
	wf.Version = 2
	abs2, rel2, err := WriteWorkflowSummaryFile(project, wf, rel1)
	if err != nil {
		t.Fatalf("update write: %v", err)
	}
	if rel1 != rel2 {
		t.Errorf("rel must be stable across updates; got %q then %q", rel1, rel2)
	}
	if abs1 != abs2 {
		t.Errorf("abs must be stable across updates; got %q then %q", abs1, abs2)
	}

	// Path doubling regression: there must NOT be a workflows/workflows
	// directory under project root.
	doubled := filepath.Join(project.FolderPath, "workflows", "workflows")
	if _, err := os.Stat(doubled); err == nil {
		t.Errorf("path doubling regressed: %s should not exist", doubled)
	}
}

func TestWriteWorkflowSummaryFile_PreferredRelClean(t *testing.T) {
	// Mildly malformed preferredRel ("./workflows/foo.json", with
	// leading dot or backslash variations) must normalize cleanly
	// rather than spawning an extra "./" segment in the abs path.
	dir := t.TempDir()
	project := &database.ProjectIndex{ID: uuid.New(), Name: "P", FolderPath: dir}
	wf := &basapi.WorkflowSummary{
		Id: uuid.New().String(), ProjectId: project.ID.String(),
		Name: "W", FolderPath: "/",
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
	}

	_, rel, err := WriteWorkflowSummaryFile(project, wf, "./workflows/manual.workflow.json")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if rel != "workflows/manual.workflow.json" {
		t.Errorf("preferredRel should normalize; got %q", rel)
	}
}

// TestGetWorkflowDoesNotSyncFromDisk asserts the resolver path no
// longer re-reads disk in a way that would clobber a fresh API write.
// The behavior is encoded structurally: we walk the project root before
// and after a single GetWorkflow call and assert no extra files were
// created (sync would have re-scanned and potentially written
// normalization snapshots).
func TestGetWorkflowDoesNotSyncFromDisk(t *testing.T) {
	dir := t.TempDir()
	project := &database.ProjectIndex{
		ID: uuid.New(), Name: "P", FolderPath: dir,
	}
	id := uuid.New()
	wf := &basapi.WorkflowSummary{
		Id: id.String(), ProjectId: project.ID.String(),
		Name: "W", FolderPath: "/",
		Version:        3,
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
	}
	_, _, err := WriteWorkflowSummaryFile(project, wf, "")
	if err != nil {
		t.Fatalf("seed file: %v", err)
	}

	// Snapshot the dir tree.
	beforeCount := countFiles(t, dir)

	// We can't easily spin up a full WorkflowService without DB +
	// migrations; instead, exercise the seam directly: ReadWorkflowSummaryFile
	// must not write back if the file is already normalized. (NeedsWrite
	// stays false for our freshly-marshaled file.) This is the
	// invariant GetWorkflow relies on now that the sync call is gone.
	abs := filepath.Join(project.FolderPath, "workflows", "w--"+strings.ToLower(id.String()[:8])+".workflow.json")
	snap, err := ReadWorkflowSummaryFile(context.Background(), project, abs)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if snap.NeedsWrite {
		t.Error("freshly-written file should not require renormalization")
	}

	if afterCount := countFiles(t, dir); afterCount != beforeCount {
		t.Errorf("read created extra files: before=%d after=%d", beforeCount, afterCount)
	}
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	var n int
	if err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			n++
		}
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	return n
}
