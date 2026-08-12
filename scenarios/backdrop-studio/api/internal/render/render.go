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
	ID, JobID, Strategy, ExecutionPath        string
	PNG                                       []byte
	Width, Height                             int
	Seed                                      int64
	TreatmentApplied                          bool
	ConditioningSubmitted, DisclosureRequired bool
	Prompt, ProvenanceJSON                    string
}

type Job struct {
	ID, StyleID, Status, ExecutionPath string
	Seed                               int64
	Candidates                         []Candidate
	SelectedCandidateID, SelectedBy    string
}

type Store struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	engine    imageengine.Executor
	generator imageengine.Generator
}

func NewStore(engine ...imageengine.Executor) *Store {
	var executor imageengine.Executor
	if len(engine) > 0 {
		executor = engine[0]
	}
	return &Store{jobs: map[string]*Job{}, engine: executor}
}

func NewStoreWithGenerator(engine imageengine.Executor, generator imageengine.Generator) *Store {
	return &Store{jobs: map[string]*Job{}, engine: engine, generator: generator}
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
		if style.Strategy == "guided" && style.Scaffold == nil {
			return Job{}, fmt.Errorf("render: guided style %q is missing its scaffold capability", style.ID)
		}
		conditioner := ""
		if style.Scaffold != nil {
			conditioner = style.Scaffold.Conditioner
		}
		result, err := scaffold.Render(scaffold.Request{Preset: preset, Conditioner: conditioner, ParamsJSON: scaffoldParams(style), Width: 320, Height: 180, Seed: candidateSeed, Regions: regions})
		if err != nil {
			return Job{}, err
		}
		if s.engine == nil {
			return Job{}, fmt.Errorf("render: image-tools executor is not configured")
		}
		input := result.PNG
		conditioningSubmitted := false
		prompt := ""
		if style.Generation != nil {
			prompt = style.Generation.PromptTemplate
		}
		if style.Strategy == "guided" || style.Strategy == "synthesized" {
			if s.generator == nil {
				return Job{}, fmt.Errorf("render: %s requires image-tools inference capability", style.Strategy)
			}
			generated, genErr := s.generator.Generate(ctx, imageengine.GenerationRequest{Prompt: prompt, Negative: generationNegative(style), Role: style.Generation.Role, Profile: style.Generation.Profile, Seed: candidateSeed, Conditioner: conditioner, Conditioning: func() []byte {
				if style.Strategy == "guided" {
					conditioningSubmitted = true
					return result.PNG
				}
				return nil
			}()})
			if genErr != nil {
				return Job{}, fmt.Errorf("render: %s inference capability: %w", style.Strategy, genErr)
			}
			if len(generated) == 0 {
				return Job{}, fmt.Errorf("render: %s inference returned an empty image", style.Strategy)
			}
			input = generated
		}
		treated, err := s.engine.Apply(ctx, input, style.Treatments, palette)
		if err != nil {
			return Job{}, fmt.Errorf("render: treatment chain: %w", err)
		}
		job.Candidates = append(job.Candidates, Candidate{ID: id(jobID, candidateSeed, i), JobID: jobID, Strategy: style.Strategy, ExecutionPath: job.ExecutionPath, PNG: treated, Width: result.Width, Height: result.Height, Seed: candidateSeed, TreatmentApplied: true, ConditioningSubmitted: conditioningSubmitted, DisclosureRequired: style.Strategy == "guided" || style.Strategy == "synthesized", Prompt: prompt, ProvenanceJSON: fmt.Sprintf(`{"style_id":%q,"strategy":%q,"seed":%d,"treatments":%q,"execution_path":%q}`, style.ID, style.Strategy, candidateSeed, style.Treatments, job.ExecutionPath)})
	}
	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()
	return copyJob(job), nil
}

func scaffoldParams(style catalog.Style) string {
	if style.Scaffold == nil {
		return ""
	}
	return style.Scaffold.ParamsJSON
}
func generationNegative(style catalog.Style) string {
	if style.Generation == nil {
		return ""
	}
	return style.Generation.Negative
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
