package golden_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"development-toolchain-validator/internal/golden"
	"development-toolchain-validator/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fakeRepo is a hand-rolled in-memory Repository used by service tests.
// Lives in this file (not a shared mocks package) so the contract is
// visible next to the tests asserting it.
type fakeRepo struct {
	mu      sync.Mutex
	byID    map[string]golden.Golden
	bySlug  map[string]string
	clock   *mocks.FakeClock
	nextID  int
	failOps map[string]error
}

func newFakeRepo(clk *mocks.FakeClock) *fakeRepo {
	return &fakeRepo{
		byID:    make(map[string]golden.Golden),
		bySlug:  make(map[string]string),
		clock:   clk,
		failOps: make(map[string]error),
	}
}

func (r *fakeRepo) Create(_ context.Context, g golden.Golden) (golden.Golden, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.failOps["create"]; err != nil {
		return golden.Golden{}, err
	}
	if _, ok := r.bySlug[g.Slug]; ok {
		return golden.Golden{}, golden.ErrGoldenAlreadyExists{Slug: g.Slug}
	}
	r.nextID++
	id := "id-" + g.Slug
	g.ID = id
	now := r.clock.Now().UTC()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	if g.LastRegeneratedAt.IsZero() {
		g.LastRegeneratedAt = g.CreatedAt
	}
	r.byID[id] = g
	r.bySlug[g.Slug] = id
	return g, nil
}

func (r *fakeRepo) Get(_ context.Context, slug string) (golden.Golden, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.bySlug[slug]
	if !ok {
		return golden.Golden{}, golden.ErrGoldenNotFound{Slug: slug}
	}
	return r.byID[id], nil
}

func (r *fakeRepo) List(_ context.Context) ([]golden.Golden, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]golden.Golden, 0, len(r.byID))
	for _, g := range r.byID {
		out = append(out, g)
	}
	return out, nil
}

func (r *fakeRepo) Update(_ context.Context, g golden.Golden) (golden.Golden, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.bySlug[g.Slug]
	if !ok {
		return golden.Golden{}, golden.ErrGoldenNotFound{Slug: g.Slug}
	}
	existing := r.byID[id]
	g.ID = existing.ID
	g.CreatedAt = existing.CreatedAt
	if g.LastRegeneratedAt.IsZero() {
		g.LastRegeneratedAt = r.clock.Now().UTC()
	}
	r.byID[id] = g
	return g, nil
}

func (r *fakeRepo) Delete(_ context.Context, slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.bySlug[slug]
	if !ok {
		return golden.ErrGoldenNotFound{Slug: slug}
	}
	delete(r.byID, id)
	delete(r.bySlug, slug)
	return nil
}

// fakeRunner records calls and returns a deterministic output. Failure
// modes are exercised by setting err.
type fakeRunner struct {
	called bool
	in     golden.RegenerateRunnerInput
	out    golden.RegenerateRunnerOutput
	err    error
}

func (f *fakeRunner) Regenerate(_ context.Context, in golden.RegenerateRunnerInput) (golden.RegenerateRunnerOutput, error) {
	f.called = true
	f.in = in
	if f.err != nil {
		return golden.RegenerateRunnerOutput{}, f.err
	}
	return f.out, nil
}

func newSvc(t *testing.T) (golden.Service, *fakeRepo, *mocks.FakeClock, *fakeRunner) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := newFakeRepo(clk)
	runner := &fakeRunner{}
	return golden.NewService(repo, clk, runner), repo, clk, runner
}

func TestService_RegisterHappyPath(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	g, err := svc.Register(context.Background(), golden.RegisterInput{
		Slug:            "reference-react-vite",
		TemplateID:      "react-vite",
		TemplateVersion: "1.0.1",
		Path:            "scenarios/reference-react-vite",
	})
	require.NoError(t, err)
	require.Equal(t, "reference-react-vite", g.Slug)
	require.Equal(t, "1.0.1", g.TemplateVersionPinned)
}

func TestService_RegisterValidationFailures(t *testing.T) {
	cases := []struct {
		name string
		in   golden.RegisterInput
		want string
	}{
		{"empty slug", golden.RegisterInput{TemplateID: "x", TemplateVersion: "1", Path: "p"}, "slug"},
		{"bad slug", golden.RegisterInput{Slug: "Bad Slug", TemplateID: "x", TemplateVersion: "1", Path: "p"}, "slug"},
		{"empty template", golden.RegisterInput{Slug: "g", TemplateVersion: "1", Path: "p"}, "template_id"},
		{"empty version", golden.RegisterInput{Slug: "g", TemplateID: "x", Path: "p"}, "template_version"},
		{"empty path", golden.RegisterInput{Slug: "g", TemplateID: "x", TemplateVersion: "1"}, "path"},
		{"absolute path", golden.RegisterInput{Slug: "g", TemplateID: "x", TemplateVersion: "1", Path: "/abs"}, "path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, _ := newSvc(t)
			_, err := svc.Register(context.Background(), tc.in)
			require.Error(t, err)
			var invalid golden.ErrInvalidGolden
			require.True(t, errors.As(err, &invalid), "expected ErrInvalidGolden, got %T: %v", err, err)
			require.Equal(t, tc.want, invalid.Field)
		})
	}
}

func TestService_RegisterDuplicateRejected(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	in := golden.RegisterInput{Slug: "g", TemplateID: "x", TemplateVersion: "1", Path: "p"}
	_, err := svc.Register(context.Background(), in)
	require.NoError(t, err)
	_, err = svc.Register(context.Background(), in)
	var exists golden.ErrGoldenAlreadyExists
	require.True(t, errors.As(err, &exists), "expected ErrGoldenAlreadyExists, got %T", err)
}

func TestService_UpdatePreservesUnsetFields(t *testing.T) {
	svc, _, clk, _ := newSvc(t)
	ctx := context.Background()
	original, err := svc.Register(ctx, golden.RegisterInput{Slug: "g", TemplateID: "react-vite", TemplateVersion: "1.0.1", Path: "scenarios/g"})
	require.NoError(t, err)

	clk.Advance(time.Hour)
	updated, err := svc.Update(ctx, golden.UpdateInput{Slug: "g", TemplateVersion: "1.0.2"})
	require.NoError(t, err)
	require.Equal(t, "1.0.2", updated.TemplateVersionPinned)
	require.Equal(t, original.Path, updated.Path, "path must stay unchanged when empty")
	require.True(t, updated.LastRegeneratedAt.After(original.LastRegeneratedAt))
}

func TestService_UpdateMissingReturnsNotFound(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	_, err := svc.Update(context.Background(), golden.UpdateInput{Slug: "ghost", Path: "p"})
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_DeleteRequiresSlug(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	err := svc.Delete(context.Background(), "")
	var invalid golden.ErrInvalidGolden
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, "slug", invalid.Field)
}

func TestService_RegenerateUpdatesVersionAndTimestamp(t *testing.T) {
	svc, _, clk, runner := newSvc(t)
	ctx := context.Background()
	_, err := svc.Register(ctx, golden.RegisterInput{Slug: "g", TemplateID: "react-vite", TemplateVersion: "1.0.1", Path: "scenarios/g"})
	require.NoError(t, err)

	runner.out = golden.RegenerateRunnerOutput{TemplateVersion: "1.0.2"}
	clk.Advance(time.Hour)

	updated, err := svc.Regenerate(ctx, "g")
	require.NoError(t, err)
	require.True(t, runner.called)
	require.Equal(t, "g", runner.in.Slug)
	require.Equal(t, "1.0.2", updated.TemplateVersionPinned)
	require.Equal(t, clk.Now(), updated.LastRegeneratedAt)
}

func TestService_RegenerateRunnerFailureWrapsError(t *testing.T) {
	svc, _, _, runner := newSvc(t)
	ctx := context.Background()
	_, err := svc.Register(ctx, golden.RegisterInput{Slug: "g", TemplateID: "react-vite", TemplateVersion: "1.0.1", Path: "scenarios/g"})
	require.NoError(t, err)

	runner.err = errors.New("boom")
	_, err = svc.Regenerate(ctx, "g")
	var rf golden.ErrRegenerateFailed
	require.True(t, errors.As(err, &rf))
	require.Equal(t, "g", rf.Slug)
}

func TestService_RegenerateMissingReturnsNotFound(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	_, err := svc.Regenerate(context.Background(), "ghost")
	var nf golden.ErrGoldenNotFound
	require.True(t, errors.As(err, &nf))
}
