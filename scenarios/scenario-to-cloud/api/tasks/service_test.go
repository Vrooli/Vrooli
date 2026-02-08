package tasks

import (
	"context"
	"testing"

	"scenario-to-cloud/domain"
)

func TestNewService_RegistersDefaultHandlers(t *testing.T) {
	svc := NewService(nil, nil, nil)
	if svc == nil {
		t.Fatal("expected service")
	}
	if svc.handlers == nil {
		t.Fatal("expected handler registry")
	}

	tests := []domain.TaskType{
		domain.TaskTypeInvestigate,
		domain.TaskTypeFix,
	}

	for _, taskType := range tests {
		h, ok := svc.handlers.Get(taskType)
		if !ok {
			t.Fatalf("expected handler for task type %s", taskType)
		}
		if got := h.TaskType(); got != taskType {
			t.Fatalf("handler task type mismatch: got %s want %s", got, taskType)
		}
	}
}

func TestHandlerRegistry_RegisterAndGet(t *testing.T) {
	reg := NewHandlerRegistry()
	if reg == nil {
		t.Fatal("expected registry")
	}

	h1 := &stubTaskHandler{taskType: domain.TaskTypeInvestigate}
	h2 := &stubTaskHandler{taskType: domain.TaskTypeInvestigate}
	reg.Register(h1)
	reg.Register(h2) // overwrite same task type

	got, ok := reg.Get(domain.TaskTypeInvestigate)
	if !ok {
		t.Fatal("expected registered handler")
	}
	if got != h2 {
		t.Fatal("expected latest handler for same task type")
	}

	if _, ok := reg.Get(domain.TaskType("unknown")); ok {
		t.Fatal("unexpected handler for unknown task type")
	}
}

type stubTaskHandler struct {
	taskType domain.TaskType
}

func (h *stubTaskHandler) TaskType() domain.TaskType {
	return h.taskType
}

func (h *stubTaskHandler) BuildPromptAndContext(_ context.Context, _ TaskInput) (PromptResult, error) {
	return PromptResult{}, nil
}

func (h *stubTaskHandler) AgentTag() string {
	return "stub"
}

func (h *stubTaskHandler) ShouldContinue(_ context.Context, _ *domain.Investigation, _ *AgentResult) (bool, string) {
	return false, ""
}
