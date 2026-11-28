//go:build audio_lab

package experiment

import (
	"fmt"
	"time"
)

// persistExecutionResult stores the terminal state and any successful output.
// The caller holds e.mu, matching the lifecycle guarantee that a concurrent
// cancel cannot publish a competing terminal state during persistence.
func (m *Manager) persistExecutionResult(base, final Experiment, result RunResult, finished time.Time) (Experiment, bool) {
	if final.Status != StatusSucceeded {
		if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
			m.logger.Printf("experiment %s: persist terminal status: %v", final.ID, err)
		}
		return final, true
	}

	if len(result.RecipeJSON) > 0 {
		final.RecipeJSON = result.RecipeJSON
	}
	if len(result.Report) == 0 {
		return m.persistSuccessfulResult(base, final, result, finished, "")
	}

	mime := result.ReportMIME
	if mime == "" {
		mime = "application/json"
	}
	key, err := m.service.storeReportBlob(m.baseCtx, final, result.Report, mime, finished)
	if err != nil {
		m.logger.Printf("experiment %s: store report: %v", final.ID, err)
		return m.persistFailure(base, finished, err.Error(), true, "report-failure")
	}
	final.ResultRef = key
	return m.persistSuccessfulResult(base, final, result, finished, key)
}

func (m *Manager) persistSuccessfulResult(base, final Experiment, result RunResult, finished time.Time, blobKey string) (Experiment, bool) {
	if err := m.service.repo.CompleteSucceeded(m.baseCtx, final, result.Runs); err != nil {
		if blobKey != "" {
			_ = m.service.blobs.Delete(m.baseCtx, blobKey)
		}
		m.logger.Printf("experiment %s: persist successful result: %v", final.ID, err)
		failed, ok := m.persistFailure(base, finished, fmt.Sprintf("persist successful result: %v", err), blobKey != "", "result-failure")
		return failed, ok
	}
	m.persistQualificationEvidence(final, result.QualificationEvidence)
	return final, true
}

func (m *Manager) persistFailure(base Experiment, finished time.Time, message string, clearResultRef bool, kind string) (Experiment, bool) {
	failed := base
	if err := failed.Transition(StatusFailed); err != nil {
		m.logger.Printf("experiment %s: transition %s status: %v", base.ID, kind, err)
		return Experiment{}, false
	}
	failed.FinishedAt = &finished
	failed.Error = message
	if clearResultRef {
		failed.ResultRef = ""
	}
	if err := m.service.repo.UpdateExperiment(m.baseCtx, failed); err != nil {
		m.logger.Printf("experiment %s: persist %s status: %v", failed.ID, kind, err)
	}
	return failed, true
}
