package renderjobs

import (
	"context"
	"errors"
	"os"
	"testing"

	"device-studio/internal/jobs"
	"device-studio/internal/projects"
)

func geometryRequest(key string) GeometryCreationRequest {
	return GeometryCreationRequest{
		ProjectID: "project", CandidateID: "candidate-1", IdempotencyKey: key,
		Brief: "compact field display", EnvelopeMM: [3]float64{80, 150, 12},
		ReferenceMediaIDs: []string{"active", "plate"},
		FundingRoute:      GeometryFundingLocal,
	}
}

func addGeometryPlate(media *integrationMedia) {
	value := media.values["active"]
	value.Kind, value.RasterKnown, value.MediaType = projects.MediaKindImage, true, "image/png"
	media.values["active"], value.ID = value, "plate"
	media.values["plate"], media.data["plate"] = value, append([]byte(nil), media.data["active"]...)
}

func submitGeometryTestJob(t *testing.T, c *Coordinator, ctx context.Context, req GeometryCreationRequest) jobs.Job {
	t.Helper()
	job, created, err := c.SubmitGeometryCreation(ctx, req)
	if err != nil || !created {
		t.Fatalf("initial geometry submit created=%v err=%v", created, err)
	}
	return job
}

func TestGeometryCreateReplayKeepsOriginalWorkerAndSourcePinsAcrossRebuild(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "terminal"}[terminal], func(t *testing.T) {
			c, ctx, workerPath := integrationCoordinator(t, &integrationOps{}, false)
			media := c.media.(*integrationMedia)
			addGeometryPlate(media)
			job := submitGeometryTestJob(t, c, ctx, geometryRequest("rebuild-replay"))
			workerPin := bindingByRole(job.Input, "worker_executable")
			oldSource := bindingByRole(job.Input, "reference_0")
			if oldSource.SHA256 != media.values["active"].SHA256 {
				t.Fatal("initial job did not pin its source bytes")
			}
			if terminal {
				cancelled, err := c.jobs.RequestCancel(ctx, job.ID, job.Version)
				if err != nil || cancelled.State != jobs.Cancelled {
					t.Fatalf("terminalize exact replay fixture: state=%s err=%v", cancelled.State, err)
				}
				job = cancelled
			}

			if err := os.WriteFile(workerPath, []byte("worker-v2-after-rebuild"), 0o700); err != nil {
				t.Fatal(err)
			}
			if terminal {
				delete(media.values, "active")
				delete(media.values, "plate")
			} else {
				changed := media.values["active"]
				changed.SHA256 = digestBytes([]byte("reference changed after initial submission"))
				media.values["active"] = changed
			}
			got, created, err := c.SubmitGeometryCreation(ctx, geometryRequest("rebuild-replay"))
			if err != nil || created {
				t.Fatalf("same semantic replay created=%v err=%v", created, err)
			}
			if got.ID != job.ID || got.State != job.State || got.InputFingerprint != job.InputFingerprint || got.Version != job.Version {
				t.Fatalf("replay changed authoritative job: got=%+v want=%+v", got, job)
			}
			if bindingByRole(got.Input, "worker_executable") != workerPin || bindingByRole(got.Input, "reference_0") != oldSource {
				t.Fatal("replay replaced the historical worker or source digest")
			}
		})
	}
}

func TestGeometryCreateSameKeyRejectsDifferentSemanticRequest(t *testing.T) {
	tests := []struct {
		name   string
		change func(*GeometryCreationRequest)
	}{
		{"project", func(r *GeometryCreationRequest) { r.ProjectID = "other-project" }},
		{"candidate", func(r *GeometryCreationRequest) { r.CandidateID = "candidate-2" }},
		{"brief", func(r *GeometryCreationRequest) { r.Brief += " with buttons" }},
		{"dimensions", func(r *GeometryCreationRequest) { r.EnvelopeMM[0]++ }},
		{"reference order", func(r *GeometryCreationRequest) {
			r.ReferenceMediaIDs[0], r.ReferenceMediaIDs[1] = r.ReferenceMediaIDs[1], r.ReferenceMediaIDs[0]
		}},
		{"funding route", func(r *GeometryCreationRequest) { r.FundingRoute = GeometryFundingBYOK }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
			addGeometryPlate(c.media.(*integrationMedia))
			_ = submitGeometryTestJob(t, c, ctx, geometryRequest("semantic-conflict"))
			changed := geometryRequest("semantic-conflict")
			tt.change(&changed)
			if _, created, err := c.SubmitGeometryCreation(ctx, changed); !errors.Is(err, jobs.ErrIdempotencyConflict) || created {
				t.Fatalf("different same-key request created=%v err=%v", created, err)
			}
		})
	}
}

func TestGeometryCreateReplayRefusesKeyOwnedByAnotherSubject(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "foreign-subject", Input: jobs.Input{
		ProjectID: "project", SubjectID: "video", SubjectKind: jobs.SubjectSourceInspection,
		RendererVersion: "source-inspection-v1", OutputProfile: "inspection-v1",
		ToolProfileDigest: digestBytes([]byte("profile")), Bindings: []jobs.ArtifactBinding{routeBinding(route)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := c.SubmitGeometryCreation(ctx, geometryRequest("foreign-subject")); !errors.Is(err, jobs.ErrIdempotencyConflict) || created {
		t.Fatalf("geometry request reused another subject's key: created=%v err=%v", created, err)
	}
}

type geometryReplayRaceJobs struct {
	jobs.Service
	hiddenKey string
	hidden    bool
}

func (s *geometryReplayRaceJobs) FindByIdempotencyKey(ctx context.Context, key string) (jobs.Job, error) {
	if key == s.hiddenKey && !s.hidden {
		s.hidden = true
		return jobs.Job{}, jobs.ErrNotFound
	}
	return s.Service.FindByIdempotencyKey(ctx, key)
}

func TestGeometryCreateRechecksSemanticReplayAfterInsertRace(t *testing.T) {
	c, ctx, workerPath := integrationCoordinator(t, &integrationOps{}, false)
	addGeometryPlate(c.media.(*integrationMedia))
	_ = submitGeometryTestJob(t, c, ctx, geometryRequest("insert-race"))
	if err := os.WriteFile(workerPath, []byte("worker-v2-after-race"), 0o700); err != nil {
		t.Fatal(err)
	}
	c.jobs = &geometryReplayRaceJobs{Service: c.jobs, hiddenKey: "insert-race"}
	got, created, err := c.SubmitGeometryCreation(ctx, geometryRequest("insert-race"))
	if err != nil || created || got.ID == "" || got.State != jobs.Queued {
		t.Fatalf("insert-race semantic readback created=%v job=%+v err=%v", created, got, err)
	}
}

type failedGeometryLookup struct{ jobs.Service }

func (failedGeometryLookup) FindByIdempotencyKey(context.Context, string) (jobs.Job, error) {
	return jobs.Job{}, errors.New("lookup failed")
}

func TestGeometryCreatePropagatesIdempotencyLookupFailure(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	c.jobs = failedGeometryLookup{c.jobs}
	if _, _, err := c.SubmitGeometryCreation(ctx, geometryRequest("lookup-error")); err == nil {
		t.Fatal("idempotency lookup failure was ignored")
	}
}
