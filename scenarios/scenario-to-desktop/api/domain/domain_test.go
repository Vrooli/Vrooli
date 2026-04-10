package domain

import (
	"encoding/json"
	"testing"
	"time"
)

// --- TaskType ---

func TestTaskType_Valid(t *testing.T) {
	tests := []struct {
		tt   TaskType
		want bool
	}{
		{TaskTypeInvestigate, true},
		{TaskTypeFix, true},
		{"unknown", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(string(tc.tt), func(t *testing.T) {
			if got := tc.tt.Valid(); got != tc.want {
				t.Errorf("TaskType(%q).Valid() = %v, want %v", tc.tt, got, tc.want)
			}
		})
	}
}

// --- InvestigationEffort ---

func TestInvestigationEffort_Valid(t *testing.T) {
	tests := []struct {
		e    InvestigationEffort
		want bool
	}{
		{EffortChecks, true},
		{EffortLogs, true},
		{EffortTrace, true},
		{"deep", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(string(tc.e), func(t *testing.T) {
			if got := tc.e.Valid(); got != tc.want {
				t.Errorf("InvestigationEffort(%q).Valid() = %v, want %v", tc.e, got, tc.want)
			}
		})
	}
}

// --- TaskFocus ---

func TestTaskFocus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		focus   TaskFocus
		wantErr bool
	}{
		{"harness only", TaskFocus{Harness: true}, false},
		{"subject only", TaskFocus{Subject: true}, false},
		{"both", TaskFocus{Harness: true, Subject: true}, false},
		{"neither", TaskFocus{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.focus.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("TaskFocus.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- FixPermissions ---

func TestFixPermissions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		perms   FixPermissions
		wantErr bool
	}{
		{"immediate only", FixPermissions{Immediate: true}, false},
		{"permanent only", FixPermissions{Permanent: true}, false},
		{"prevention only", FixPermissions{Prevention: true}, false},
		{"all", FixPermissions{Immediate: true, Permanent: true, Prevention: true}, false},
		{"none", FixPermissions{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.perms.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("FixPermissions.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestFixPermissions_String(t *testing.T) {
	tests := []struct {
		name  string
		perms FixPermissions
		want  string
	}{
		{"none", FixPermissions{}, "none"},
		{"immediate", FixPermissions{Immediate: true}, "immediate"},
		{"permanent", FixPermissions{Permanent: true}, "permanent"},
		{"prevention", FixPermissions{Prevention: true}, "prevention"},
		{"immediate and permanent", FixPermissions{Immediate: true, Permanent: true}, "immediate,permanent"},
		{"all", FixPermissions{Immediate: true, Permanent: true, Prevention: true}, "immediate,permanent,prevention"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.perms.String(); got != tc.want {
				t.Errorf("FixPermissions.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- CreateTaskRequest.Validate ---

func TestCreateTaskRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateTaskRequest
		wantErr string // empty means no error
	}{
		{
			name:    "empty pipeline ID",
			req:     CreateTaskRequest{TaskType: TaskTypeInvestigate, Focus: TaskFocus{Harness: true}},
			wantErr: "pipeline_id is required",
		},
		{
			name:    "invalid task type",
			req:     CreateTaskRequest{PipelineID: "p1", TaskType: "bad", Focus: TaskFocus{Harness: true}},
			wantErr: "invalid task_type",
		},
		{
			name:    "no focus selected",
			req:     CreateTaskRequest{PipelineID: "p1", TaskType: TaskTypeInvestigate},
			wantErr: "at least one focus",
		},
		{
			name: "investigate defaults effort to logs",
			req: CreateTaskRequest{
				PipelineID: "p1",
				TaskType:   TaskTypeInvestigate,
				Focus:      TaskFocus{Harness: true},
			},
		},
		{
			name: "investigate invalid effort",
			req: CreateTaskRequest{
				PipelineID: "p1",
				TaskType:   TaskTypeInvestigate,
				Focus:      TaskFocus{Harness: true},
				Effort:     "deep",
			},
			wantErr: "invalid effort level",
		},
		{
			name: "fix requires permissions",
			req: CreateTaskRequest{
				PipelineID: "p1",
				TaskType:   TaskTypeFix,
				Focus:      TaskFocus{Subject: true},
			},
			wantErr: "at least one permission",
		},
		{
			name: "fix max iterations exceeds limit",
			req: CreateTaskRequest{
				PipelineID:    "p1",
				TaskType:      TaskTypeFix,
				Focus:         TaskFocus{Subject: true},
				Permissions:   FixPermissions{Immediate: true},
				MaxIterations: 11,
			},
			wantErr: "max_iterations cannot exceed 10",
		},
		{
			name: "fix defaults max iterations to 5",
			req: CreateTaskRequest{
				PipelineID:  "p1",
				TaskType:    TaskTypeFix,
				Focus:       TaskFocus{Subject: true},
				Permissions: FixPermissions{Immediate: true},
			},
		},
		{
			name: "valid investigate with all fields",
			req: CreateTaskRequest{
				PipelineID: "p1",
				TaskType:   TaskTypeInvestigate,
				Focus:      TaskFocus{Harness: true, Subject: true},
				Effort:     EffortTrace,
				Note:       "some context",
			},
		},
		{
			name: "valid fix with all fields",
			req: CreateTaskRequest{
				PipelineID:    "p1",
				TaskType:      TaskTypeFix,
				Focus:         TaskFocus{Harness: true},
				Permissions:   FixPermissions{Immediate: true, Permanent: true},
				MaxIterations: 7,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Check defaults were applied
				if tc.req.TaskType == TaskTypeInvestigate && tc.req.Effort == "" {
					t.Error("expected effort to be defaulted")
				}
				if tc.req.TaskType == TaskTypeFix && tc.req.MaxIterations <= 0 {
					t.Error("expected max_iterations to be defaulted")
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got := err.Error(); !contains(got, tc.wantErr) {
					t.Errorf("error = %q, want to contain %q", got, tc.wantErr)
				}
			}
		})
	}
}

func TestCreateTaskRequest_Validate_DefaultsEffort(t *testing.T) {
	req := &CreateTaskRequest{
		PipelineID: "p1",
		TaskType:   TaskTypeInvestigate,
		Focus:      TaskFocus{Harness: true},
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.Effort != EffortLogs {
		t.Errorf("expected effort %q, got %q", EffortLogs, req.Effort)
	}
}

func TestCreateTaskRequest_Validate_DefaultsMaxIterations(t *testing.T) {
	req := &CreateTaskRequest{
		PipelineID:  "p1",
		TaskType:    TaskTypeFix,
		Focus:       TaskFocus{Subject: true},
		Permissions: FixPermissions{Permanent: true},
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.MaxIterations != 5 {
		t.Errorf("expected MaxIterations 5, got %d", req.MaxIterations)
	}
}

// --- Investigation ---

func TestInvestigation_ToSummary(t *testing.T) {
	findings := "some findings"
	now := time.Now()
	inv := &Investigation{
		ID:          "inv-1",
		PipelineID:  "p-1",
		Status:      InvestigationStatusCompleted,
		Findings:    &findings,
		Progress:    100,
		CreatedAt:   now,
		CompletedAt: &now,
	}

	summary := inv.ToSummary()

	if summary.ID != "inv-1" {
		t.Errorf("ID = %q, want %q", summary.ID, "inv-1")
	}
	if !summary.HasFindings {
		t.Error("expected HasFindings = true")
	}
	if summary.Status != InvestigationStatusCompleted {
		t.Errorf("Status = %q, want %q", summary.Status, InvestigationStatusCompleted)
	}
}

func TestInvestigation_ToSummary_NoFindings(t *testing.T) {
	inv := &Investigation{
		ID:         "inv-2",
		PipelineID: "p-2",
		Status:     InvestigationStatusRunning,
		Progress:   50,
		CreatedAt:  time.Now(),
	}
	summary := inv.ToSummary()
	if summary.HasFindings {
		t.Error("expected HasFindings = false when Findings is nil")
	}
}

func TestInvestigation_ToSummary_EmptyFindings(t *testing.T) {
	empty := ""
	inv := &Investigation{
		ID:       "inv-3",
		Findings: &empty,
	}
	summary := inv.ToSummary()
	if summary.HasFindings {
		t.Error("expected HasFindings = false when Findings is empty string")
	}
}

func TestInvestigation_ToSummary_ExtractsSourceInvestigationID(t *testing.T) {
	details := InvestigationDetails{
		Source:                "agent-manager",
		SourceInvestigationID: "orig-inv-1",
	}
	detailsJSON, _ := json.Marshal(details)

	inv := &Investigation{
		ID:      "inv-4",
		Details: detailsJSON,
	}
	summary := inv.ToSummary()
	if summary.SourceInvestigationID == nil || *summary.SourceInvestigationID != "orig-inv-1" {
		t.Errorf("expected SourceInvestigationID = %q, got %v", "orig-inv-1", summary.SourceInvestigationID)
	}
}

func TestInvestigation_ParseDetails(t *testing.T) {
	t.Run("nil details", func(t *testing.T) {
		inv := &Investigation{}
		details, err := inv.ParseDetails()
		if err != nil {
			t.Fatal(err)
		}
		if details != nil {
			t.Error("expected nil details")
		}
	})

	t.Run("valid details", func(t *testing.T) {
		d := InvestigationDetails{
			Source:        "agent-manager",
			OperationMode: "report-only",
			TriggerReason: "user_requested",
			FailedStage:   "build",
		}
		data, _ := json.Marshal(d)
		inv := &Investigation{Details: data}

		parsed, err := inv.ParseDetails()
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Source != "agent-manager" {
			t.Errorf("Source = %q, want %q", parsed.Source, "agent-manager")
		}
		if parsed.FailedStage != "build" {
			t.Errorf("FailedStage = %q, want %q", parsed.FailedStage, "build")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		inv := &Investigation{Details: json.RawMessage(`{bad`)}
		_, err := inv.ParseDetails()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestInvestigation_SetDetails(t *testing.T) {
	t.Run("nil details clears field", func(t *testing.T) {
		inv := &Investigation{Details: json.RawMessage(`{"source":"test"}`)}
		if err := inv.SetDetails(nil); err != nil {
			t.Fatal(err)
		}
		if inv.Details != nil {
			t.Error("expected nil Details after SetDetails(nil)")
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		inv := &Investigation{}
		d := &InvestigationDetails{
			Source:        "agent-manager",
			OperationMode: "fix-application",
			TokensUsed:    1500,
		}
		if err := inv.SetDetails(d); err != nil {
			t.Fatal(err)
		}

		parsed, err := inv.ParseDetails()
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Source != d.Source {
			t.Errorf("Source = %q, want %q", parsed.Source, d.Source)
		}
		if parsed.TokensUsed != d.TokensUsed {
			t.Errorf("TokensUsed = %d, want %d", parsed.TokensUsed, d.TokensUsed)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
