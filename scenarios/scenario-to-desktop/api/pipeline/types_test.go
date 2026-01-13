package pipeline

import (
	"testing"
)

func TestConfigGetStopOnFailure(t *testing.T) {
	t.Run("nil returns true", func(t *testing.T) {
		c := Config{}
		if !c.GetStopOnFailure() {
			t.Error("expected true for nil StopOnFailure")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		b := true
		c := Config{StopOnFailure: &b}
		if !c.GetStopOnFailure() {
			t.Error("expected true")
		}
	})

	t.Run("explicit false", func(t *testing.T) {
		b := false
		c := Config{StopOnFailure: &b}
		if c.GetStopOnFailure() {
			t.Error("expected false")
		}
	})
}

func TestConfigGetDeploymentMode(t *testing.T) {
	t.Run("empty returns bundled", func(t *testing.T) {
		c := Config{}
		if c.GetDeploymentMode() != "bundled" {
			t.Errorf("expected 'bundled', got %q", c.GetDeploymentMode())
		}
	})

	t.Run("explicit value", func(t *testing.T) {
		c := Config{DeploymentMode: "proxy"}
		if c.GetDeploymentMode() != "proxy" {
			t.Errorf("expected 'proxy', got %q", c.GetDeploymentMode())
		}
	})
}

func TestConfigGetTemplateType(t *testing.T) {
	t.Run("empty returns basic", func(t *testing.T) {
		c := Config{}
		if c.GetTemplateType() != "basic" {
			t.Errorf("expected 'basic', got %q", c.GetTemplateType())
		}
	})

	t.Run("explicit value", func(t *testing.T) {
		c := Config{TemplateType: "advanced"}
		if c.GetTemplateType() != "advanced" {
			t.Errorf("expected 'advanced', got %q", c.GetTemplateType())
		}
	})
}

func TestStageResultIsComplete(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusCompleted, true},
		{StatusFailed, true},
		{StatusSkipped, true},
		{StatusPending, false},
		{StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			r := StageResult{Status: tt.status}
			if r.IsComplete() != tt.expected {
				t.Errorf("IsComplete() = %v, expected %v", r.IsComplete(), tt.expected)
			}
		})
	}
}

func TestStageResultIsSuccess(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusCompleted, true},
		{StatusSkipped, true},
		{StatusFailed, false},
		{StatusPending, false},
		{StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			r := StageResult{Status: tt.status}
			if r.IsSuccess() != tt.expected {
				t.Errorf("IsSuccess() = %v, expected %v", r.IsSuccess(), tt.expected)
			}
		})
	}
}

func TestStatusProgress(t *testing.T) {
	t.Run("empty stage order", func(t *testing.T) {
		s := Status{StageOrder: []string{}}
		if s.Progress() != 0 {
			t.Errorf("expected 0, got %f", s.Progress())
		}
	})

	t.Run("no completed stages", func(t *testing.T) {
		s := Status{
			StageOrder: []string{"a", "b", "c"},
			Stages: map[string]*StageResult{
				"a": {Status: StatusRunning},
			},
		}
		if s.Progress() != 0 {
			t.Errorf("expected 0, got %f", s.Progress())
		}
	})

	t.Run("all stages completed", func(t *testing.T) {
		s := Status{
			StageOrder: []string{"a", "b"},
			Stages: map[string]*StageResult{
				"a": {Status: StatusCompleted},
				"b": {Status: StatusCompleted},
			},
		}
		if s.Progress() != 1.0 {
			t.Errorf("expected 1.0, got %f", s.Progress())
		}
	})

	t.Run("half completed", func(t *testing.T) {
		s := Status{
			StageOrder: []string{"a", "b", "c", "d"},
			Stages: map[string]*StageResult{
				"a": {Status: StatusCompleted},
				"b": {Status: StatusSkipped},
				"c": {Status: StatusRunning},
			},
		}
		if s.Progress() != 0.5 {
			t.Errorf("expected 0.5, got %f", s.Progress())
		}
	})

	t.Run("failed stage counts as complete", func(t *testing.T) {
		s := Status{
			StageOrder: []string{"a", "b"},
			Stages: map[string]*StageResult{
				"a": {Status: StatusFailed},
			},
		}
		if s.Progress() != 0.5 {
			t.Errorf("expected 0.5, got %f", s.Progress())
		}
	})
}

func TestStatusIsComplete(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusCompleted, true},
		{StatusFailed, true},
		{StatusCancelled, true},
		{StatusPending, false},
		{StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			s := Status{Status: tt.status}
			if s.IsComplete() != tt.expected {
				t.Errorf("IsComplete() = %v, expected %v", s.IsComplete(), tt.expected)
			}
		})
	}
}
