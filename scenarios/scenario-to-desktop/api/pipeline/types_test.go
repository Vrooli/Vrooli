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

func TestStatusCanResume(t *testing.T) {
	tests := []struct {
		name              string
		status            string
		stoppedAfterStage string
		expected          bool
	}{
		{
			name:              "completed with stopped stage can resume",
			status:            StatusCompleted,
			stoppedAfterStage: "preflight",
			expected:          true,
		},
		{
			name:              "completed without stopped stage cannot resume",
			status:            StatusCompleted,
			stoppedAfterStage: "",
			expected:          false,
		},
		{
			name:              "failed cannot resume",
			status:            StatusFailed,
			stoppedAfterStage: "preflight",
			expected:          false,
		},
		{
			name:              "running cannot resume",
			status:            StatusRunning,
			stoppedAfterStage: "",
			expected:          false,
		},
		{
			name:              "cancelled cannot resume",
			status:            StatusCancelled,
			stoppedAfterStage: "generate",
			expected:          false,
		},
		{
			name:              "pending cannot resume",
			status:            StatusPending,
			stoppedAfterStage: "",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Status{
				Status:            tt.status,
				StoppedAfterStage: tt.stoppedAfterStage,
			}
			if s.CanResume() != tt.expected {
				t.Errorf("CanResume() = %v, expected %v", s.CanResume(), tt.expected)
			}
		})
	}
}

func TestStatusGetNextResumeStage(t *testing.T) {
	tests := []struct {
		name              string
		status            string
		stoppedAfterStage string
		expected          string
	}{
		{
			name:              "stopped after bundle returns preflight",
			status:            StatusCompleted,
			stoppedAfterStage: StageBundle,
			expected:          StagePreflight,
		},
		{
			name:              "stopped after preflight returns generate",
			status:            StatusCompleted,
			stoppedAfterStage: StagePreflight,
			expected:          StageGenerate,
		},
		{
			name:              "stopped after generate returns build",
			status:            StatusCompleted,
			stoppedAfterStage: StageGenerate,
			expected:          StageBuild,
		},
		{
			name:              "stopped after build returns smoketest",
			status:            StatusCompleted,
			stoppedAfterStage: StageBuild,
			expected:          StageSmokeTest,
		},
		{
			name:              "stopped after smoketest returns distribution",
			status:            StatusCompleted,
			stoppedAfterStage: StageSmokeTest,
			expected:          StageDistribution,
		},
		{
			name:              "stopped after distribution returns empty (last stage)",
			status:            StatusCompleted,
			stoppedAfterStage: StageDistribution,
			expected:          "",
		},
		{
			name:              "not resumable returns empty",
			status:            StatusFailed,
			stoppedAfterStage: StagePreflight,
			expected:          "",
		},
		{
			name:              "no stopped stage returns empty",
			status:            StatusCompleted,
			stoppedAfterStage: "",
			expected:          "",
		},
		{
			name:              "invalid stopped stage returns empty",
			status:            StatusCompleted,
			stoppedAfterStage: "invalid-stage",
			expected:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Status{
				Status:            tt.status,
				StoppedAfterStage: tt.stoppedAfterStage,
			}
			if got := s.GetNextResumeStage(); got != tt.expected {
				t.Errorf("GetNextResumeStage() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestIsValidStageName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{StageBundle, true},
		{StagePreflight, true},
		{StageGenerate, true},
		{StageBuild, true},
		{StageSmokeTest, true},
		{StageDistribution, true},
		{"invalid", false},
		{"", false},
		{"Bundle", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidStageName(tt.name); got != tt.expected {
				t.Errorf("IsValidStageName(%q) = %v, expected %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestConfigGetStopAfterStage(t *testing.T) {
	t.Run("empty returns empty", func(t *testing.T) {
		c := Config{}
		if c.GetStopAfterStage() != "" {
			t.Errorf("expected empty, got %q", c.GetStopAfterStage())
		}
	})

	t.Run("returns set value", func(t *testing.T) {
		c := Config{StopAfterStage: "generate"}
		if c.GetStopAfterStage() != "generate" {
			t.Errorf("expected 'generate', got %q", c.GetStopAfterStage())
		}
	})
}

func TestConfigGetResumeFromStage(t *testing.T) {
	t.Run("empty returns empty", func(t *testing.T) {
		c := Config{}
		if c.GetResumeFromStage() != "" {
			t.Errorf("expected empty, got %q", c.GetResumeFromStage())
		}
	})

	t.Run("returns set value", func(t *testing.T) {
		c := Config{ResumeFromStage: "build"}
		if c.GetResumeFromStage() != "build" {
			t.Errorf("expected 'build', got %q", c.GetResumeFromStage())
		}
	})
}
