package notes_test

import (
	"testing"
	"{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

func TestAttachmentUploadWorkflow_TransitionMatrix(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.AssertTransitionMatrix(
		t,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		formalAttachmentUploadMatrix(t, artifact),
		func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
			next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
			return next.Status, err
		},
	)
}

func TestAttachmentUploadWorkflow_ReplaysTraces(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.ReplayTraces(t, formalAttachmentUploadTraces(t, artifact), func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
		next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
		return next.Status, err
	})
}

func TestAttachmentUploadWorkflow_ReplaysFormalModelArtifacts(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{
		ContractPath:  "api/internal/notes/attachment_upload_workflow.flow.json",
		ModelPath:     "api/internal/notes/attachment_upload_workflow.qnt",
		GeneratorPath: "tools/temporal-model/generate.mjs",
		Invariants: []string{
			"TypeOK",
			"TerminalClosure",
			"IllegalTransitionsPreserveState",
			"AllDeclaredTransitionsCovered",
			"MetadataRequiresBytesStored",
		},
	})
	transition := func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
		next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
		return next.Status, err
	}
	modeltest.AssertFormalTransitionsReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		transition,
	)
	modeltest.AssertFormalTracesReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		transition,
	)
}

func TestAttachmentUploadWorkflow_RejectsUnknownState(t *testing.T) {
	err := notes.CheckAttachmentUploadInvariants(notes.AttachmentUploadState{Status: "ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown attachment upload status")
}

func formalAttachmentUploadMatrix(t *testing.T, artifact modeltest.FormalArtifact) []modeltest.MatrixRow[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent] {
	t.Helper()
	statusByName := map[string]notes.AttachmentUploadStatus{
		string(notes.AttachmentUploadReceived):         notes.AttachmentUploadReceived,
		string(notes.AttachmentUploadBytesStored):      notes.AttachmentUploadBytesStored,
		string(notes.AttachmentUploadMetadataRecorded): notes.AttachmentUploadMetadataRecorded,
		string(notes.AttachmentUploadFailed):           notes.AttachmentUploadFailed,
	}
	eventByName := map[string]notes.AttachmentUploadEvent{
		string(notes.AttachmentUploadStoreBytes):     notes.AttachmentUploadStoreBytes,
		string(notes.AttachmentUploadRecordMetadata): notes.AttachmentUploadRecordMetadata,
		string(notes.AttachmentUploadFail):           notes.AttachmentUploadFail,
	}
	rows := make([]modeltest.MatrixRow[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent], 0, len(artifact.Transitions))
	for _, transition := range artifact.Transitions {
		rows = append(rows, modeltest.MatrixRow[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
			Name:    transition.From + "/" + transition.Event,
			From:    statusByName[transition.From],
			Event:   eventByName[transition.Event],
			To:      statusByName[transition.To],
			WantErr: transition.WantError,
		})
	}
	return rows
}

func formalAttachmentUploadTraces(t *testing.T, artifact modeltest.FormalArtifact) []modeltest.Trace[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent] {
	t.Helper()
	statusByName := map[string]notes.AttachmentUploadStatus{
		string(notes.AttachmentUploadReceived):         notes.AttachmentUploadReceived,
		string(notes.AttachmentUploadBytesStored):      notes.AttachmentUploadBytesStored,
		string(notes.AttachmentUploadMetadataRecorded): notes.AttachmentUploadMetadataRecorded,
		string(notes.AttachmentUploadFailed):           notes.AttachmentUploadFailed,
	}
	eventByName := map[string]notes.AttachmentUploadEvent{
		string(notes.AttachmentUploadStoreBytes):     notes.AttachmentUploadStoreBytes,
		string(notes.AttachmentUploadRecordMetadata): notes.AttachmentUploadRecordMetadata,
		string(notes.AttachmentUploadFail):           notes.AttachmentUploadFail,
	}
	formalTraces := append(artifact.NamedTraces, artifact.GeneratedTraces...)
	traces := make([]modeltest.Trace[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent], 0, len(formalTraces))
	for _, trace := range formalTraces {
		steps := make([]modeltest.TraceStep[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent], 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, modeltest.TraceStep[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
				Event:   eventByName[step.Event],
				Want:    statusByName[step.Want],
				WantErr: step.WantError,
			})
		}
		traces = append(traces, modeltest.Trace[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
			Name:    trace.Name,
			Initial: statusByName[trace.Initial],
			Steps:   steps,
		})
	}
	return traces
}
