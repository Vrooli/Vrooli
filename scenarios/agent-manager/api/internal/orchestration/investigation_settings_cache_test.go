package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/promptmanager"
)

type countingSkillReader struct {
	mu      sync.Mutex
	reads   map[string]int
	content map[string]string
	err     error
}

func (c *countingSkillReader) ReadSkill(_ context.Context, id string, _ map[string]string, _ bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads[id]++
	if c.err != nil {
		return "", c.err
	}
	return c.content[id], nil
}

func (c *countingSkillReader) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, v := range c.reads {
		n += v
	}
	return n
}

type memInvestigationSettings struct{ settings *domain.InvestigationSettings }

func (m *memInvestigationSettings) Get(context.Context) (*domain.InvestigationSettings, error) {
	cp := *m.settings
	return &cp, nil
}
func (m *memInvestigationSettings) Update(_ context.Context, s *domain.InvestigationSettings) error {
	cp := *s
	m.settings = &cp
	return nil
}
func (m *memInvestigationSettings) Reset(context.Context) error {
	m.settings = domain.DefaultInvestigationSettings()
	return nil
}

func TestGetInvestigationSettingsServesPromptsFromCacheUntilExpiryOrInvalidation(t *testing.T) {
	reader := &countingSkillReader{reads: map[string]int{}, content: map[string]string{
		"agent-manager-process-investigation":       "investigate v1",
		"agent-manager-process-investigation-apply": "apply v1",
	}}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	o := New(nil, nil, nil,
		WithPromptClient(reader),
		WithInvestigationSettings(&memInvestigationSettings{settings: domain.DefaultInvestigationSettings()}),
		WithClock(func() time.Time { return now }),
	)
	ctx := context.Background()

	for i := 0; i < 50; i++ {
		s, err := o.GetInvestigationSettings(ctx)
		if err != nil || s.PromptTemplate != "investigate v1" || s.ApplyPromptTemplate != "apply v1" {
			t.Fatalf("call %d: settings=%+v err=%v", i, s, err)
		}
	}
	if got := reader.total(); got != 2 {
		t.Fatalf("50 gets read prompt-manager %d times, want 2 (one per template)", got)
	}

	now = now.Add(investigationPromptTTL + time.Second)
	reader.content["agent-manager-process-investigation"] = "investigate v2"
	if s, _ := o.GetInvestigationSettings(ctx); s.PromptTemplate != "investigate v2" {
		t.Fatalf("expired cache served %q, want a fresh read", s.PromptTemplate)
	}
	if got := reader.total(); got != 4 {
		t.Fatalf("reads after expiry=%d, want 4", got)
	}

	reader.content["agent-manager-process-investigation"] = "investigate v3"
	if err := o.UpdateInvestigationSettings(ctx, domain.DefaultInvestigationSettings()); err != nil {
		t.Fatalf("update: %v", err)
	}
	if s, _ := o.GetInvestigationSettings(ctx); s.PromptTemplate != "investigate v3" {
		t.Fatalf("after update served %q, want invalidation to force a fresh read", s.PromptTemplate)
	}

	reader.content["agent-manager-process-investigation"] = "investigate v4"
	if err := o.ResetInvestigationSettings(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if s, _ := o.GetInvestigationSettings(ctx); s.PromptTemplate != "investigate v4" {
		t.Fatalf("after reset served %q, want invalidation to force a fresh read", s.PromptTemplate)
	}
}

func TestGetInvestigationSettingsKeepsLastPromptsAndBacksOffWhenPromptManagerFails(t *testing.T) {
	reader := &countingSkillReader{reads: map[string]int{}, content: map[string]string{
		"agent-manager-process-investigation":       "investigate v1",
		"agent-manager-process-investigation-apply": "apply v1",
	}}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	o := New(nil, nil, nil,
		WithPromptClient(reader),
		WithInvestigationSettings(&memInvestigationSettings{settings: domain.DefaultInvestigationSettings()}),
		WithClock(func() time.Time { return now }),
	)
	ctx := context.Background()
	if _, err := o.GetInvestigationSettings(ctx); err != nil {
		t.Fatal(err)
	}

	now = now.Add(investigationPromptTTL + time.Second)
	reader.err = errors.New("prompt-manager down")
	before := reader.total()
	for i := 0; i < 20; i++ {
		s, err := o.GetInvestigationSettings(ctx)
		if err != nil || s.PromptTemplate != "investigate v1" || s.ApplyPromptTemplate != "apply v1" {
			t.Fatalf("outage call %d: settings=%+v err=%v", i, s, err)
		}
	}
	if got := reader.total() - before; got != 2 {
		t.Fatalf("20 gets during outage read prompt-manager %d times, want 2", got)
	}
}

func TestGetInvestigationSettingsBacksOffOnColdStartFailure(t *testing.T) {
	reader := &countingSkillReader{reads: map[string]int{}, err: errors.New("prompt-manager down")}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	o := New(nil, nil, nil,
		WithPromptClient(reader),
		WithInvestigationSettings(&memInvestigationSettings{settings: domain.DefaultInvestigationSettings()}),
		WithClock(func() time.Time { return now }),
	)
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		if _, err := o.GetInvestigationSettings(ctx); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := reader.total(); got != 2 {
		t.Fatalf("20 cold-start gets read prompt-manager %d times, want 2", got)
	}

	now = now.Add(investigationPromptRetryTTL + time.Second)
	reader.err = nil
	reader.content = map[string]string{
		"agent-manager-process-investigation":       "investigate v1",
		"agent-manager-process-investigation-apply": "apply v1",
	}
	if s, _ := o.GetInvestigationSettings(ctx); s.PromptTemplate != "investigate v1" {
		t.Fatalf("after the retry interval served %q, want a fresh read", s.PromptTemplate)
	}
}

// partialAdmin lets the first skill update succeed and the second fail.
type partialAdmin struct {
	*countingSkillReader
	updated []string
}

func (p *partialAdmin) UpdateSkill(_ context.Context, id string, patch promptmanager.PromptSkillUpdate) (promptmanager.PromptSkill, error) {
	if len(p.updated) == 1 {
		return promptmanager.PromptSkill{}, errors.New("second update failed")
	}
	p.updated = append(p.updated, id)
	p.content[id] = *patch.Content
	return promptmanager.PromptSkill{}, nil
}
func (p *partialAdmin) ListSkills(context.Context, string) ([]promptmanager.PromptSkill, error) {
	return nil, nil
}
func (p *partialAdmin) GetSkill(context.Context, string) (promptmanager.PromptSkill, error) {
	return promptmanager.PromptSkill{}, nil
}
func (p *partialAdmin) GetSkillVersions(context.Context, string) (promptmanager.PromptSkillVersions, error) {
	return promptmanager.PromptSkillVersions{}, nil
}
func (p *partialAdmin) RevertSkillVersion(context.Context, string, int) error { return nil }

func TestUpdateInvestigationSettingsInvalidatesWhenSecondSkillUpdateFails(t *testing.T) {
	admin := &partialAdmin{countingSkillReader: &countingSkillReader{reads: map[string]int{}, content: map[string]string{
		"agent-manager-process-investigation":       "investigate v1",
		"agent-manager-process-investigation-apply": "apply v1",
	}}}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	o := New(nil, nil, nil,
		WithPromptClient(admin),
		WithInvestigationSettings(&memInvestigationSettings{settings: domain.DefaultInvestigationSettings()}),
		WithClock(func() time.Time { return now }),
	)
	ctx := context.Background()
	if _, err := o.GetInvestigationSettings(ctx); err != nil {
		t.Fatal(err)
	}

	next := domain.DefaultInvestigationSettings()
	next.PromptTemplate, next.ApplyPromptTemplate = "investigate v2", "apply v2"
	if err := o.UpdateInvestigationSettings(ctx, next); err == nil {
		t.Fatal("update should report the second skill failure")
	}
	if s, _ := o.GetInvestigationSettings(ctx); s.PromptTemplate != "investigate v2" {
		t.Fatalf("after a partial update served %q, want the first skill's new content", s.PromptTemplate)
	}
}

func TestGetInvestigationSettingsReadsOnceUnderConcurrencyAndDoesNotBlockOnTheRead(t *testing.T) {
	gate := make(chan struct{})
	reader := &gatedSkillReader{gate: gate, entered: make(chan struct{}, 8)}
	o := New(nil, nil, nil,
		WithPromptClient(reader),
		WithInvestigationSettings(&memInvestigationSettings{settings: domain.DefaultInvestigationSettings()}),
	)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _, _ = o.GetInvestigationSettings(ctx) }()
	<-reader.entered // the first caller is inside the (blocked) read

	// Other callers must not queue behind it.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			_, _ = o.GetInvestigationSettings(ctx)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("callers blocked behind an in-flight prompt-manager read")
	}
	close(gate)
	wg.Wait()
	if got := reader.reads(); got != 2 {
		t.Fatalf("concurrent gets read prompt-manager %d times, want 2", got)
	}
}

type gatedSkillReader struct {
	gate    chan struct{}
	entered chan struct{}
	mu      sync.Mutex
	n       int
}

func (g *gatedSkillReader) ReadSkill(context.Context, string, map[string]string, bool) (string, error) {
	g.mu.Lock()
	g.n++
	g.mu.Unlock()
	g.entered <- struct{}{}
	<-g.gate
	return "content", nil
}

func (g *gatedSkillReader) reads() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.n
}
