package queue

import "time"

// DiagnosticsAPI defines the interface for queue processor diagnostics.
// This interface enables testing and provides clear boundaries for diagnostic operations.
// The Processor implements this interface directly, providing testability
// without requiring full extraction of the tightly-coupled diagnostic logic.
type DiagnosticsAPI interface {
	// GetResumeDiagnostics returns information about blockers that will be
	// cleared when resuming the processor.
	GetResumeDiagnostics() ResumeDiagnostics

	// ResetForResume performs recovery work (terminating processes, clearing
	// rate limits, reconciling queue state) so resuming starts from a clean slate.
	ResetForResume() ResumeResetSummary

	// ResumeWithReset resumes queue processing after performing safety recovery steps.
	ResumeWithReset() ResumeResetSummary

	// ResumeWithoutReset restarts processing without terminating running
	// executions or clearing state.
	ResumeWithoutReset()

	// GetQueueStatus returns current queue processor status and metrics.
	GetQueueStatus() map[string]any

	// GetLastProcessedTime returns the timestamp of the last queue processing cycle.
	GetLastProcessedTime() time.Time
}

// Verify Processor implements DiagnosticsAPI at compile time.
var _ DiagnosticsAPI = (*Processor)(nil)
