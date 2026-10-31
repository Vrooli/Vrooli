package stagedmessages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"sort"

	"connectrpc.com/connect"
	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *Service) continueDraft(ctx context.Context, repo, dir string, req *v1.StartStagedMessageRequest) (*v1.StagedMessageJob, error) {
	previous, err := s.store.Get(ctx, req.ContinuationOfJobId)
	if err != nil {
		return nil, err
	}
	if previous.Operation != operation || previous.RepositoryID != repo || previous.State != reviewjobs.Succeeded || previous.ResultJSON == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("a completed partial draft from this repository is required"))
	}
	var input DraftInput
	if err = json.Unmarshal([]byte(previous.InputJSON), &input); err != nil {
		return nil, err
	}
	prior := &v1.StagedMessageSuggestion{}
	if err = protojson.Unmarshal([]byte(previous.ResultJSON), prior); err != nil {
		return nil, err
	}
	limited := prior.MessageScope == v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED || (prior.MessageScope == v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_UNSPECIFIED && prior.GetCoverage().GetPartial())
	if input.Snapshot == nil || req.SnapshotDigest != input.Snapshot.Digest || !limited {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("continuation requires the original partial staged subject"))
	}
	if req.CollectionId != "" && req.CollectionId != previous.ResultRef {
		return nil, ErrConflict
	}
	// Selected sources are inherited exactly; continuation does not silently
	// replace them or recollect a different index under the original digest.
	selected := append([]string(nil), req.SelectedContextIds...)
	want := make([]string, 0, len(input.Contexts))
	for _, source := range input.Contexts {
		want = append(want, source.Id)
	}
	sort.Strings(selected)
	sort.Strings(want)
	if len(selected) > 0 && !slices.Equal(selected, want) {
		return nil, ErrConflict
	}
	input.FormatVersion = 4
	input.ContinuationOfJobID = previous.ID
	input.PriorResult = prior
	if req.OperatorNote != "" && req.OperatorNote != input.OperatorNote {
		return nil, ErrConflict
	}
	if existing, err := s.store.GetByIdempotency(ctx, key(repo, req.RequestKey)); err == nil {
		var retained DraftInput
		if json.Unmarshal([]byte(existing.InputJSON), &retained) != nil || retained.ContinuationOfJobID != previous.ID || retained.OperatorNote != input.OperatorNote || existing.RepositoryID != repo || existing.Operation != operation {
			return nil, ErrConflict
		}
		return s.read(ctx, repo, dir, existing, "get", 0)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	standing, err := s.Check(ctx, repo, dir, input.Snapshot.Digest)
	if err != nil {
		return nil, err
	}
	if !standing.Applicable {
		return nil, ErrConflict
	}
	input.JobID = id("message-")
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, errors.New("continuation immutable input exceeds durable bound")
	}
	job, _, err := s.store.Create(ctx, reviewjobs.Job{ID: input.JobID, IdempotencyKey: key(repo, req.RequestKey), InputDigest: digest(raw), InputJSON: string(raw), ResultRef: previous.ResultRef, Operation: operation, RepositoryID: repo, PolicyVersion: "staged-message-v1", State: reviewjobs.Queued})
	if err != nil {
		return nil, err
	}
	var actual DraftInput
	if json.Unmarshal([]byte(job.InputJSON), &actual) != nil || actual.ContinuationOfJobID != previous.ID || actual.OperatorNote != input.OperatorNote {
		return nil, ErrConflict
	}
	return s.read(ctx, repo, dir, job, "get", 0)
}
