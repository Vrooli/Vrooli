package codingjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type service struct {
	repo Repository
	now  Clock
}

func NewService(repo Repository, now Clock) Service {
	if repo == nil || now == nil {
		panic("codingjobs.NewService requires repository and clock")
	}
	return &service{repo: repo, now: now}
}

func (s *service) Create(ctx context.Context, in CreateInput) (Job, error) {
	in.CaseID = strings.TrimSpace(in.CaseID)
	in.ReplayKey = strings.TrimSpace(in.ReplayKey)
	in.Project = strings.TrimSpace(in.Project)
	in.Title = strings.TrimSpace(in.Title)
	in.Task = strings.TrimSpace(in.Task)
	in.SourceSealRef = strings.TrimSpace(in.SourceSealRef)
	if in.ReplayKey == "" || in.CaseID == "" || in.Project == "" || in.Title == "" || in.Task == "" || in.SourceSealRef == "" {
		return Job{}, ErrInvalid{Reason: "replay_key, case_id, project, title, task, and source_seal_ref are required"}
	}
	if len(in.ReplayKey) > 256 || len(in.CaseID) > 128 || len(in.Project) > 256 || len(in.Title) > 512 || len(in.Task) > 32768 || len(in.SourceSealRef) > 512 {
		return Job{}, ErrInvalid{Reason: "one or more coding job fields exceed their size limit"}
	}
	now := s.now().UTC()
	job := Job{ID: uuid.NewString(), ReplayKey: in.ReplayKey, CaseID: in.CaseID, Project: in.Project, Title: in.Title, Task: in.Task, State: StateReady, SourceSealRef: in.SourceSealRef, Revision: 1, CreatedAt: now, UpdatedAt: now,
		Events: []Event{{Sequence: 1, Stage: "job_created", Actor: "client-rehearsal", EvidenceRef: in.SourceSealRef, OccurredAt: now}}}
	return s.repo.Create(ctx, job)
}

func (s *service) RecordStage(ctx context.Context, in StageInput) (Job, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Stage = strings.TrimSpace(in.Stage)
	in.Actor = strings.TrimSpace(in.Actor)
	in.EvidenceRef = strings.TrimSpace(in.EvidenceRef)
	in.EvidenceSHA256 = strings.ToLower(strings.TrimSpace(in.EvidenceSHA256))
	in.Note = strings.TrimSpace(in.Note)
	if in.ID == "" || in.ExpectedRevision <= 0 {
		return Job{}, ErrInvalid{Reason: "id and positive expected_revision are required"}
	}
	job, err := s.repo.Get(ctx, in.ID)
	if err != nil {
		return Job{}, err
	}
	if job.Revision != in.ExpectedRevision {
		return Job{}, ErrConflict{Reason: "coding job changed; reload its current revision"}
	}
	if len(job.Events) >= 32 {
		return Job{}, ErrConflict{Reason: "coding job event limit reached"}
	}
	if len(in.Note) > 1024 || len(in.EvidenceRef) > 512 {
		return Job{}, ErrInvalid{Reason: "event evidence exceeds its size limit"}
	}
	if in.EvidenceSHA256 != "" && !validSHA256(in.EvidenceSHA256) {
		return Job{}, ErrInvalid{Reason: "evidence_sha256 must be 64 hexadecimal characters"}
	}
	next, err := transition(job.State, in.Stage, in.Actor, in.EvidenceRef, in.EvidenceSHA256)
	if err != nil {
		return Job{}, err
	}
	now := s.now().UTC()
	job.State = next
	if in.Stage == "solver_completed" {
		job.SolverRunRef = in.EvidenceRef
	}
	job.Revision++
	job.UpdatedAt = now
	job.Events = append(job.Events, Event{Sequence: uint64(len(job.Events) + 1), Stage: in.Stage, Actor: in.Actor, EvidenceRef: in.EvidenceRef, EvidenceSHA256: in.EvidenceSHA256, Note: in.Note, OccurredAt: now})
	return s.repo.Update(ctx, job, in.ExpectedRevision)
}

func transition(state State, stage, actor, ref, digest string) (State, error) {
	if ref == "" {
		return "", ErrInvalid{Reason: "evidence_ref is required for stage transitions"}
	}
	switch {
	case state == StateReady && stage == "solver_started" && actor == "agent-manager-solver":
		return StateSolving, nil
	case state == StateSolving && stage == "solver_completed" && actor == "agent-manager-solver" && validSHA256(digest):
		return StateScoring, nil
	case state == StateScoring && stage == "scoring_started" && actor == "agent-manager-scorer":
		return StateScoring, nil
	default:
		return "", ErrConflict{Reason: fmt.Sprintf("stage %q by %q is not allowed while coding job is %q", stage, actor, state)}
	}
}

// CandidateSealDigest binds source review to the durable solver-completion
// event. Callers should pass a job loaded from the repository and should only
// expose the returned digest through Workspace Sandbox's candidate-only read
// contract; never accept a digest directly from a browser request.
func CandidateSealDigest(job Job) (string, error) {
	if job.ID == "" || job.CaseID == "" || job.State != StateReviewable {
		return "", ErrConflict{Reason: "candidate source is available only for a reviewable coding job"}
	}
	var found *Event
	for i := range job.Events {
		event := &job.Events[i]
		if event.Stage != "solver_completed" {
			continue
		}
		if found != nil {
			return "", ErrConflict{Reason: "coding job has multiple solver completion events"}
		}
		found = event
	}
	if found == nil || found.Actor != "agent-manager-solver" || found.EvidenceRef == "" || found.EvidenceRef != job.SolverRunRef || !validSHA256(found.EvidenceSHA256) {
		return "", ErrConflict{Reason: "coding job has no valid durable candidate seal binding"}
	}
	return strings.ToLower(found.EvidenceSHA256), nil
}

func (s *service) RecordEvaluation(ctx context.Context, in EvaluationInput) (Job, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.ReceiptRef = strings.TrimSpace(in.ReceiptRef)
	in.ReceiptSHA256 = strings.ToLower(strings.TrimSpace(in.ReceiptSHA256))
	in.Score = strings.TrimSpace(in.Score)
	if in.ID == "" || in.ReceiptRef == "" || in.ExpectedRevision <= 0 || !validSHA256(in.ReceiptSHA256) {
		return Job{}, ErrInvalid{Reason: "id, receipt_ref, receipt_sha256, and positive expected_revision are required"}
	}
	if in.Score != "strict_pass" && in.Score != "candidate_failed" && in.Score != "reference_failed" && in.Score != "preimage_not_red" && in.Score != "invalid" {
		return Job{}, ErrInvalid{Reason: "score must be a closed three-way evaluation outcome"}
	}
	job, err := s.repo.Get(ctx, in.ID)
	if err != nil {
		return Job{}, err
	}
	if job.Revision != in.ExpectedRevision {
		return Job{}, ErrConflict{Reason: "coding job changed; reload its current revision"}
	}
	if job.State != StateScoring || job.EvaluatorReceiptRef != "" {
		return Job{}, ErrConflict{Reason: "only one evaluation may be recorded after scoring begins"}
	}
	now := s.now().UTC()
	job.EvaluatorReceiptRef = in.ReceiptRef
	job.EvaluatorReceiptSHA256 = in.ReceiptSHA256
	job.Score = in.Score
	job.State = StateFailed
	if in.Score == "strict_pass" {
		job.State = StateReviewable
	}
	job.Revision++
	job.UpdatedAt = now
	job.Events = append(job.Events, Event{Sequence: uint64(len(job.Events) + 1), Stage: "evaluation_recorded", Actor: "test-genie", EvidenceRef: in.ReceiptRef, EvidenceSHA256: in.ReceiptSHA256, Note: in.Score, OccurredAt: now})
	return s.repo.Update(ctx, job, in.ExpectedRevision)
}

func (s *service) Get(ctx context.Context, id string) (Job, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *service) List(ctx context.Context, limit int) ([]Job, error) {
	return s.repo.List(ctx, limit)
}

func validSHA256(v string) bool {
	if len(v) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}

type ErrInvalid struct{ Reason string }

func (e ErrInvalid) Error() string { return e.Reason }

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return "coding job not found: " + e.ID }

type ErrConflict struct{ Reason string }

func (e ErrConflict) Error() string { return e.Reason }
