package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func TestWorkflowPersistenceCallersUseSafeAtomicReplacement(t *testing.T) {
	project := &database.ProjectIndex{ID: uuid.New(), Name: "Atomic", FolderPath: t.TempDir()}
	workflowID := uuid.New()
	newWorkflow := func(description string) *basapi.WorkflowSummary {
		return &basapi.WorkflowSummary{
			Id: workflowID.String(), ProjectId: project.ID.String(), Name: "Atomic flow",
			Description: description, FolderPath: "/", Version: 1,
			FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
		}
	}

	tests := []struct {
		name   string
		target string
		write  func(description string) error
	}{
		{
			name:   "workflow version snapshot",
			target: workflowVersionFilePath(project.FolderPath, workflowID, 1),
			write: func(description string) error {
				return persistWorkflowVersionSnapshot(project, newWorkflow(description))
			},
		},
		{
			name:   "converted workflow",
			target: filepath.Join(project.FolderPath, "workflows", "atomic-flow--"+shortID(workflowID)+workflowFileExt),
			write: func(description string) error {
				_, _, err := WriteConvertedWorkflow(project, &ConvertResult{Workflow: newWorkflow(description), SourcePath: "workflows/source.json"})
				return err
			},
		},
		{
			name:   "in-place workflow",
			target: filepath.Join(project.FolderPath, "workflows", "in-place.workflow.json"),
			write: func(description string) error {
				return WriteWorkflowInPlace(filepath.Join(project.FolderPath, "workflows", "in-place.workflow.json"), newWorkflow(description))
			},
		},
		{
			name:   "project metadata",
			target: projectMetadataPath(project.FolderPath),
			write:  func(description string) error { return persistProjectProto(project, description) },
		},
		{
			name:   "workflow summary",
			target: filepath.Join(project.FolderPath, "workflows", "atomic.workflow.json"),
			write: func(description string) error {
				_, _, err := WriteWorkflowSummaryFile(project, newWorkflow(description), "workflows/atomic.workflow.json")
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const writers = 12
			start := make(chan struct{})
			errs := make(chan error, writers)
			var wg sync.WaitGroup
			for i := 0; i < writers; i++ {
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					errs <- tc.write(fmt.Sprintf("writer-%02d", i))
				}()
			}
			close(start)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					t.Fatalf("concurrent writer failed: %v", err)
				}
			}

			data, err := os.ReadFile(tc.target)
			if err != nil {
				t.Fatalf("read committed file: %v", err)
			}
			if !json.Valid(data) {
				t.Fatalf("committed file is not complete JSON: %s", data)
			}
			if _, err := os.Stat(tc.target + ".tmp"); !os.IsNotExist(err) {
				t.Fatalf("fixed temporary file remains: %v", err)
			}
		})
	}
}
