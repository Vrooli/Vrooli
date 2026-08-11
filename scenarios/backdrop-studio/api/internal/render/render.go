package render

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scaffold"
)

type Candidate struct {
	ID, JobID, Strategy, ExecutionPath string
	PNG                                []byte
	Width, Height                      int
	Seed                               int64
	TreatmentApplied                   bool
}

type Job struct {
	ID, StyleID, Status, ExecutionPath string
	Seed                               int64
	Candidates                         []Candidate
	SelectedCandidateID, SelectedBy    string
}

type Store struct {
	mu     sync.RWMutex
	jobs   map[string]*Job
	engine imageengine.Executor
}

func NewStore(engine ...imageengine.Executor) *Store {
	var executor imageengine.Executor
	if len(engine) > 0 {
		executor = engine[0]
	}
	return &Store{jobs: map[string]*Job{}, engine: executor}
}

func (s *Store) Submit(style catalog.Style, placement string, seed int64, count int) (Job, error) {
	return s.SubmitWithContext(context.Background(), style, placement, seed, count, nil)
}

func (s *Store) SubmitWithContext(ctx context.Context, style catalog.Style, placement string, seed int64, count int, palette map[string]string) (Job, error) {
	if style.ID == "" || style.Strategy == "" {
		return Job{}, fmt.Errorf("render: style is required")
	}
	if placement != "" && !contains(style.Placements, placement) {
		return Job{}, fmt.Errorf("render: placement %q is not permitted by style %q", placement, style.ID)
	}
	if count <= 0 {
		count = 1
	}
	if count > 16 {
		return Job{}, fmt.Errorf("render: candidate_count must be between 1 and 16")
	}
	jobID := id(style.ID, seed, count)
	job := &Job{ID: jobID, StyleID: style.ID, Status: "completed", Seed: seed, ExecutionPath: expectedPath(style.Strategy)}
	for i := 0; i < count; i++ {
		candidateSeed := seed + int64(i)
		preset := style.Subject
		switch preset {
		case "statuary_architecture":
			preset = "arcade"
		case "geological":
			preset = "terrain"
		case "non_representational":
			preset = "field"
		case "horizon":
		default:
			preset = "field"
		}
		regions := make([]scaffold.Region, 0, len(style.Regions))
		for _, region := range style.Regions {
			regions = append(regions, scaffold.Region{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height})
		}
		result, err := scaffold.Render(scaffold.Request{Preset: preset, Width: 320, Height: 180, Seed: candidateSeed, Regions: regions})
		if err != nil {
			return Job{}, err
		}
		if s.engine == nil {
			return Job{}, fmt.Errorf("render: image-tools executor is not configured")
		}
		treated, err := s.engine.Apply(ctx, result.PNG, style.Treatments, palette)
		if err != nil {
			return Job{}, fmt.Errorf("render: treatment chain: %w", err)
		}
		job.Candidates = append(job.Candidates, Candidate{ID: id(jobID, candidateSeed, i), JobID: jobID, Strategy: style.Strategy, ExecutionPath: job.ExecutionPath, PNG: treated, Width: result.Width, Height: result.Height, Seed: candidateSeed, TreatmentApplied: true})
	}
	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()
	return copyJob(job), nil
}

func (s *Store) Get(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("render: job %q not found", id)
	}
	return copyJob(job), nil
}
func (s *Store) Select(jobID, candidateID, actor string) (Job, error) {
	if actor == "" {
		return Job{}, fmt.Errorf("render: actor is required for selection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return Job{}, fmt.Errorf("render: job %q not found", jobID)
	}
	for _, candidate := range job.Candidates {
		if candidate.ID == candidateID {
			job.SelectedCandidateID = candidateID
			job.SelectedBy = actor
			job.Status = "selected"
			return copyJob(job), nil
		}
	}
	return Job{}, fmt.Errorf("render: candidate %q does not belong to job %q", candidateID, jobID)
}
func (s *Store) Candidate(jobID, candidateID string) (Candidate, error) {
	job, err := s.Get(jobID)
	if err != nil {
		return Candidate{}, err
	}
	for _, c := range job.Candidates {
		if c.ID == candidateID {
			return c, nil
		}
	}
	return Candidate{}, fmt.Errorf("render: candidate %q not found", candidateID)
}

func expectedPath(strategy string) string {
	switch strategy {
	case "guided":
		return "scaffold → image-tools inference → treatment"
	case "synthesized":
		return "image-tools inference → treatment"
	default:
		return "procedural → treatment"
	}
}
func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
func id(parts ...interface{}) string {
	h := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(h, "%v\x00", part)
	}
	return hex.EncodeToString(h.Sum(nil))[:20]
}
func copyJob(job *Job) Job {
	out := *job
	out.Candidates = make([]Candidate, len(job.Candidates))
	for i, c := range job.Candidates {
		out.Candidates[i] = c
		out.Candidates[i].PNG = append([]byte(nil), c.PNG...)
	}
	return out
}
