package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"device-studio/internal/jobs"
)

func typedInput() jobs.Input {
	in := input()
	in.ShotRevisionID, in.ShotRevision, in.SourceClockDigest = "", 0, ""
	in.SubjectKind = jobs.SubjectSourceInspection
	in.SubjectID = "media-opaque-id"
	return in
}

func TestGeometrySubjectDataIsBoundedAndCanonical(t *testing.T) {
	in := typedInput()
	in.SubjectKind = jobs.SubjectGeometryCreate
	in.SubjectData = json.RawMessage(`{ "candidate_id": "device-1", "references": ["media-1"] }`)
	in.SubjectID = "device-1"
	canonical, first, err := jobs.CanonicalInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical.SubjectData) != `{"candidate_id":"device-1","references":["media-1"]}` {
		t.Fatalf("subject payload was not canonicalized: %s", canonical.SubjectData)
	}
	in.SubjectData = json.RawMessage(`{"candidate_id":"device-1","references":["media-1"]}`)
	_, second, err := jobs.CanonicalInput(in)
	if err != nil || first != second {
		t.Fatalf("equivalent payload fingerprint changed: %q %q %v", first, second, err)
	}
}

func TestGeometrySubjectDataRejectsInvalidOrUnboundedPayload(t *testing.T) {
	for name, payload := range map[string]json.RawMessage{
		"invalid JSON": json.RawMessage(`{"candidate_id":`),
		"oversized":    json.RawMessage(`{"value":"` + strings.Repeat("x", 64<<10) + `"}`),
	} {
		t.Run(name, func(t *testing.T) {
			in := typedInput()
			in.SubjectKind, in.SubjectID, in.SubjectData = jobs.SubjectGeometryCreate, "device-1", payload
			if _, _, err := jobs.CanonicalInput(in); !errors.Is(err, jobs.ErrInvalid) {
				t.Fatalf("expected invalid input refusal, got %v", err)
			}
		})
	}
	in := typedInput()
	in.SubjectData = json.RawMessage(`{"unexpected":true}`)
	if _, _, err := jobs.CanonicalInput(in); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("non-geometry subjects must refuse subject data, got %v", err)
	}
}

func TestLegacyD10StillInputFingerprintRemainsByteCompatible(t *testing.T) {
	const worker = "a0e9ded721851b5517252927627dd37369a2487f0ebf09337e1048e8d7030267"
	const routeSHA = "1d4fc6f670d3491a5b6f16fdfcfbde697ad01cdef383f0e5e4516fd003412394"
	in := jobs.Input{
		ProjectID:      "5a340161-1061-4096-b9dd-76a4ecd9b943",
		ShotRevisionID: "shot:a8665db4-4940-48ad-b4e4-50b1b2443cc3:revision:1", ShotRevision: 1,
		Bindings: []jobs.ArtifactBinding{
			{Role: "_internal_storage_route", ArtifactID: "route-sha256:" + routeSHA, RevisionID: "test", SHA256: routeSHA},
			{Role: "active_mask", ArtifactID: "dc2caa42-73de-40f3-bca5-56b3bf70aafc", RevisionID: "sha256:63b8751df790f9ad2145a1ee7fc9320473d8da682025c7ab5636c421a553dcf2", SHA256: "63b8751df790f9ad2145a1ee7fc9320473d8da682025c7ab5636c421a553dcf2"},
			{Role: "hardware_mask", ArtifactID: "8d8e47d7-c4a9-4731-a62c-8664d5b8ae70", RevisionID: "sha256:a122e0ec5193cb45a7cb3660715d0fbec04938bb69f64ce5bdf942c3b5260470", SHA256: "a122e0ec5193cb45a7cb3660715d0fbec04938bb69f64ce5bdf942c3b5260470"},
			{Role: "plate", ArtifactID: "47e9b5b5-8ff6-4c66-856e-f883c75b638a", RevisionID: "sha256:9d1c223677a610b9e63c13ccba51c8a5d44f87d0d087fc5e977e58897ceccf4a", SHA256: "9d1c223677a610b9e63c13ccba51c8a5d44f87d0d087fc5e977e58897ceccf4a"},
			{Role: "render_closure", ArtifactID: "renderjobs/closures/fb053f77476b979153653cb92d535889a025a93b8668ea89c10dff1c2636c006.json", RevisionID: "sha256:fb053f77476b979153653cb92d535889a025a93b8668ea89c10dff1c2636c006", SHA256: "fb053f77476b979153653cb92d535889a025a93b8668ea89c10dff1c2636c006"},
			{Role: "screen", ArtifactID: "ef7b06ae-31ce-4515-bd2e-ca2f8b7af6b3", RevisionID: "sha256:bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d", SHA256: "bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d"},
			{Role: "shot_snapshot", ArtifactID: "shot:a8665db4-4940-48ad-b4e4-50b1b2443cc3:revision:1", RevisionID: "revision:1", SHA256: "c5619bd8ea457b391eb1f0d635ec5fec8372d619d0dcc064b2bc9d71eff57b2d"},
			{Role: "source_active_mask", ArtifactID: "renderjobs/inputs/63b8751df790f9ad2145a1ee7fc9320473d8da682025c7ab5636c421a553dcf2/active_mask", RevisionID: "sha256:63b8751df790f9ad2145a1ee7fc9320473d8da682025c7ab5636c421a553dcf2", SHA256: "63b8751df790f9ad2145a1ee7fc9320473d8da682025c7ab5636c421a553dcf2"},
			{Role: "source_hardware_mask", ArtifactID: "renderjobs/inputs/a122e0ec5193cb45a7cb3660715d0fbec04938bb69f64ce5bdf942c3b5260470/hardware_mask", RevisionID: "sha256:a122e0ec5193cb45a7cb3660715d0fbec04938bb69f64ce5bdf942c3b5260470", SHA256: "a122e0ec5193cb45a7cb3660715d0fbec04938bb69f64ce5bdf942c3b5260470"},
			{Role: "source_plate", ArtifactID: "renderjobs/inputs/9d1c223677a610b9e63c13ccba51c8a5d44f87d0d087fc5e977e58897ceccf4a/plate", RevisionID: "sha256:9d1c223677a610b9e63c13ccba51c8a5d44f87d0d087fc5e977e58897ceccf4a", SHA256: "9d1c223677a610b9e63c13ccba51c8a5d44f87d0d087fc5e977e58897ceccf4a"},
			{Role: "source_screen", ArtifactID: "renderjobs/inputs/bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d/screen", RevisionID: "sha256:bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d", SHA256: "bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d"},
			{Role: "worker_executable", ArtifactID: "renderjobs/workers/" + worker + "/device-studio-api", RevisionID: "sha256:" + worker, SHA256: worker},
		},
		SourceClockDigest: "613ab59abf008796a69cd188936b2c7c4422178910d508348ebc8f5166e2a83b",
		ToolProfileDigest: "3e727b6ab7e28463f819affd67ad22428da2793d0946f8e8813394006e31a8f1",
		RendererVersion:   "go-still-presentation-v2", OutputProfile: "png-native-v1",
	}
	canonical, fingerprint, err := jobs.CanonicalInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != "6800978f88c8f05f0d7226efd6305e70e4ff5f72bfd11f4642167a2c5b7ed256" {
		t.Fatalf("D10 canonical fingerprint changed: %s", fingerprint)
	}
	if canonical.SubjectKind != "" || canonical.SubjectID != "" {
		t.Fatalf("legacy canonical form gained subject fields: %+v", canonical)
	}
}

func TestNewSubjectKindsAreExplicitAndUnknownKindsReject(t *testing.T) {
	if _, _, err := jobs.CanonicalInput(typedInput()); err != nil {
		t.Fatalf("valid inspection input rejected: %v", err)
	}
	bad := typedInput()
	bad.SubjectKind = jobs.SubjectKind("source_media_inspection.v9")
	if _, _, err := jobs.CanonicalInput(bad); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("unknown subject accepted: %v", err)
	}
	missing := typedInput()
	missing.SubjectID = ""
	if _, _, err := jobs.CanonicalInput(missing); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("subject without identity accepted: %v", err)
	}
}

func TestFailBeforeDispatchIsAtomicAndRetryWaitsForFinalization(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "pre-dispatch.db"))
	svc := newService(repo)
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "pre-dispatch", Input: typedInput()})
	if err != nil {
		t.Fatal(err)
	}
	attempt := job.Attempts[0]
	failed, err := svc.FailBeforeDispatch(context.Background(), job.ID, attempt.ID, "invalid immutable manifest", job.Version)
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != jobs.Failed || failed.Attempts[0].State != jobs.AttemptFailed || failed.Attempts[0].Execution != nil || failed.Attempts[0].ExecutionReleasedAt != nil || failed.Attempts[0].FinishedAt == nil {
		t.Fatalf("pre-dispatch failure fabricated execution evidence or missed terminal state: %+v", failed)
	}
	if _, err := svc.PendingSubjectFinalization(context.Background(), job.ID); err != nil {
		t.Fatalf("terminal projection marker was not committed with rejection: %v", err)
	}
	if _, _, err := svc.RetrySourceInspection(context.Background(), job.ID, failed.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("retry passed before terminal projection acknowledgement: %v", err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CompleteSubjectFinalization(context.Background(), job.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	retried, created, err := svc.RetrySourceInspection(context.Background(), job.ID, failed.Version)
	if err != nil || !created || retried.State != jobs.Queued || len(retried.Attempts) != 2 || retried.Attempts[0].Execution != nil || retried.Attempts[1].ID == attempt.ID {
		t.Fatalf("retry after acknowledged pre-dispatch failure: created=%v job=%+v err=%v", created, retried, err)
	}
}

func TestTypedSubjectTerminalOutcomesHaveRetryableFinalization(t *testing.T) {
	tests := []struct {
		name   string
		finish func(t *testing.T, svc jobs.Service, job jobs.Job)
	}{
		{name: "queued cancellation", finish: func(t *testing.T, svc jobs.Service, job jobs.Job) {
			if _, err := svc.RequestCancel(context.Background(), job.ID, job.Version); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "known failure", finish: func(t *testing.T, svc jobs.Service, job jobs.Job) {
			a := job.Attempts[0]
			running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: a.ID, AttemptNumber: a.Number, OwnerID: "test-owner", ScopeID: "test-scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Fail(context.Background(), job.ID, a.ID, "known rejection", running.Version); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "success", finish: func(t *testing.T, svc jobs.Service, job jobs.Job) {
			a := job.Attempts[0]
			running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: a.ID, AttemptNumber: a.Number, OwnerID: "test-owner", ScopeID: "test-scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
			if err != nil {
				t.Fatal(err)
			}
			c := outputs()
			c.AttemptID = a.ID
			c.ExecutionClaimID = running.Attempts[0].Execution.ID
			c.InputFingerprint = running.InputFingerprint
			if _, err = svc.Complete(context.Background(), job.ID, a.ID, c, running.Version); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := openRepo(t, filepath.Join(t.TempDir(), "subject.db"))
			svc := newService(repo)
			job, created, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "typed-" + tc.name, Input: typedInput()})
			if err != nil || !created {
				t.Fatalf("submit: created=%v err=%v", created, err)
			}
			tc.finish(t, svc, job)
			pending, err := svc.PendingSubjectFinalization(context.Background(), job.ID)
			if err != nil || pending.JobID != job.ID || pending.AttemptID != job.Attempts[0].ID || pending.Kind != jobs.SubjectSourceInspection {
				t.Fatalf("pending terminal projection: %+v err=%v", pending, err)
			}
			rows, err := svc.ListProgress(context.Background(), jobs.Cursor{}, 10)
			if err != nil || len(rows) != 1 || rows[0].ID != job.ID {
				t.Fatalf("pending finalization not progress-visible: rows=%d err=%v", len(rows), err)
			}
			// Retrying the projection does not consume its marker until the owner confirms success.
			if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); err != nil {
				t.Fatal(err)
			}
			if err = svc.CompleteSubjectFinalization(context.Background(), job.ID, pending.AttemptID, pending.Kind); err != nil {
				t.Fatal(err)
			}
			terminal, err := svc.Get(context.Background(), job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if terminal.Attempts[0].Execution != nil && terminal.Attempts[0].ExecutionReleasedAt == nil {
				if _, err = svc.RecordReleased(context.Background(), job.ID, pending.AttemptID, terminal.Version, terminal.UpdatedAt); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); !errors.Is(err, jobs.ErrNotFound) {
				t.Fatalf("completed projection remains pending: %v", err)
			}
			rows, err = svc.ListProgress(context.Background(), jobs.Cursor{}, 10)
			if err != nil || len(rows) != 0 {
				t.Fatalf("completed projection remains in progress: rows=%d err=%v", len(rows), err)
			}
		})
	}
}
