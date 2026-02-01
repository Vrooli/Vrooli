package smoketest

import "testing"

func TestOutputParser_ParseResult(t *testing.T) {
	config := DefaultConfig()
	parser := NewOutputParser(config)

	tests := []struct {
		name                     string
		output                   string
		wantPassed               bool
		wantTelemetryUploaded    bool
		wantTelemetryUploadError bool
	}{
		{
			name:                     "empty output",
			output:                   "",
			wantPassed:               false,
			wantTelemetryUploaded:    false,
			wantTelemetryUploadError: false,
		},
		{
			name:                     "success only",
			output:                   "Starting app...\nSMOKE_TEST_RESULT=passed\nDone",
			wantPassed:               true,
			wantTelemetryUploaded:    false,
			wantTelemetryUploadError: false,
		},
		{
			name:                     "success with telemetry uploaded",
			output:                   "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=ok",
			wantPassed:               true,
			wantTelemetryUploaded:    true,
			wantTelemetryUploadError: false,
		},
		{
			name:                     "success with telemetry error",
			output:                   "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=error",
			wantPassed:               true,
			wantTelemetryUploaded:    false,
			wantTelemetryUploadError: true,
		},
		{
			name:                     "telemetry uploaded without success",
			output:                   "SMOKE_TEST_UPLOAD=ok",
			wantPassed:               false,
			wantTelemetryUploaded:    true,
			wantTelemetryUploadError: false,
		},
		{
			name:                     "telemetry error only",
			output:                   "SMOKE_TEST_UPLOAD=error",
			wantPassed:               false,
			wantTelemetryUploaded:    false,
			wantTelemetryUploadError: true,
		},
		{
			name:                     "both telemetry markers present",
			output:                   "SMOKE_TEST_UPLOAD=ok\nSMOKE_TEST_UPLOAD=error",
			wantPassed:               false,
			wantTelemetryUploaded:    true,
			wantTelemetryUploadError: true,
		},
		{
			name:                     "markers embedded in other text",
			output:                   "log: SMOKE_TEST_RESULT=passed happened\ntelemetry: SMOKE_TEST_UPLOAD=ok done",
			wantPassed:               true,
			wantTelemetryUploaded:    true,
			wantTelemetryUploadError: false,
		},
		{
			name:                     "partial marker not matched",
			output:                   "SMOKE_TEST_RESULT=fail\nSMOKE_TEST_UPLOAD=pending",
			wantPassed:               false,
			wantTelemetryUploaded:    false,
			wantTelemetryUploadError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseResult(tt.output)

			if result.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", result.Passed, tt.wantPassed)
			}
			if result.TelemetryUploaded != tt.wantTelemetryUploaded {
				t.Errorf("TelemetryUploaded = %v, want %v", result.TelemetryUploaded, tt.wantTelemetryUploaded)
			}
			if result.TelemetryUploadError != tt.wantTelemetryUploadError {
				t.Errorf("TelemetryUploadError = %v, want %v", result.TelemetryUploadError, tt.wantTelemetryUploadError)
			}
		})
	}
}

func TestOutputParser_CustomConfig(t *testing.T) {
	config := Config{
		SuccessMarker:       "TEST_OK",
		UploadSuccessMarker: "UPLOAD_OK",
		UploadErrorMarker:   "UPLOAD_FAIL",
	}
	parser := NewOutputParser(config)

	tests := []struct {
		name   string
		output string
		want   OutputResult
	}{
		{
			name:   "custom success marker",
			output: "TEST_OK",
			want:   OutputResult{Passed: true, Warnings: []string{}},
		},
		{
			name:   "custom upload success marker",
			output: "UPLOAD_OK",
			want:   OutputResult{TelemetryUploaded: true, Warnings: []string{}},
		},
		{
			name:   "custom upload error marker",
			output: "UPLOAD_FAIL",
			want:   OutputResult{TelemetryUploadError: true, Warnings: []string{}},
		},
		{
			name:   "default markers not matched",
			output: "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=ok",
			want:   OutputResult{Warnings: []string{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseResult(tt.output)
			if result.Passed != tt.want.Passed {
				t.Errorf("Passed = %v, want %v", result.Passed, tt.want.Passed)
			}
			if result.TelemetryUploaded != tt.want.TelemetryUploaded {
				t.Errorf("TelemetryUploaded = %v, want %v", result.TelemetryUploaded, tt.want.TelemetryUploaded)
			}
			if result.TelemetryUploadError != tt.want.TelemetryUploadError {
				t.Errorf("TelemetryUploadError = %v, want %v", result.TelemetryUploadError, tt.want.TelemetryUploadError)
			}
		})
	}
}

func TestOutputParser_EnhancedMarkers(t *testing.T) {
	config := DefaultConfig()
	parser := NewOutputParser(config)

	tests := []struct {
		name             string
		output           string
		wantInitComplete bool
		wantCleanExit    bool
		wantWarningCount int
	}{
		{
			name:             "full sequence",
			output:           "SMOKE_TEST_INIT=started\nSMOKE_TEST_READY=true\nSMOKE_TEST_RESULT=passed\nSMOKE_TEST_EXIT=clean",
			wantInitComplete: true,
			wantCleanExit:    true,
			wantWarningCount: 0,
		},
		{
			name:             "success without init or exit",
			output:           "SMOKE_TEST_RESULT=passed",
			wantInitComplete: false,
			wantCleanExit:    false,
			wantWarningCount: 2, // missing init and exit warnings
		},
		{
			name:             "init without success",
			output:           "SMOKE_TEST_INIT=started",
			wantInitComplete: true,
			wantCleanExit:    false,
			wantWarningCount: 0, // no warnings because no success
		},
		{
			name:             "success with init but no exit",
			output:           "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
			wantInitComplete: true,
			wantCleanExit:    false,
			wantWarningCount: 1, // missing exit warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseResult(tt.output)
			if result.InitComplete != tt.wantInitComplete {
				t.Errorf("InitComplete = %v, want %v", result.InitComplete, tt.wantInitComplete)
			}
			if result.CleanShutdown != tt.wantCleanExit {
				t.Errorf("CleanShutdown = %v, want %v", result.CleanShutdown, tt.wantCleanExit)
			}
			if len(result.Warnings) != tt.wantWarningCount {
				t.Errorf("Warnings count = %d, want %d; warnings: %v", len(result.Warnings), tt.wantWarningCount, result.Warnings)
			}
		})
	}
}

func TestOutputParser_ValidateSequence(t *testing.T) {
	config := DefaultConfig()
	parser := NewOutputParser(config)

	tests := []struct {
		name                string
		output              string
		wantValid           bool
		wantStageCount      int
		wantMissingCount    int
		wantOutOfOrderCount int
	}{
		{
			name:                "full correct sequence",
			output:              "SMOKE_TEST_INIT=started\nSMOKE_TEST_READY=true\nSMOKE_TEST_RESULT=passed\nSMOKE_TEST_EXIT=clean",
			wantValid:           true,
			wantStageCount:      4,
			wantMissingCount:    0,
			wantOutOfOrderCount: 0,
		},
		{
			name:                "minimal valid sequence",
			output:              "SMOKE_TEST_RESULT=passed",
			wantValid:           true,
			wantStageCount:      1,
			wantMissingCount:    0, // init, ready, exit are optional
			wantOutOfOrderCount: 0,
		},
		{
			name:                "out of order - passed before init",
			output:              "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_INIT=started",
			wantValid:           false,
			wantStageCount:      2,
			wantMissingCount:    0,
			wantOutOfOrderCount: 1,
		},
		{
			name:                "out of order - exit before passed",
			output:              "SMOKE_TEST_INIT=started\nSMOKE_TEST_EXIT=clean\nSMOKE_TEST_RESULT=passed",
			wantValid:           false,
			wantStageCount:      3,
			wantMissingCount:    0,
			wantOutOfOrderCount: 1,
		},
		{
			name:                "empty output",
			output:              "",
			wantValid:           false, // passed is required
			wantStageCount:      0,
			wantMissingCount:    1, // missing "passed"
			wantOutOfOrderCount: 0,
		},
		{
			name:                "duplicate markers - first wins",
			output:              "SMOKE_TEST_INIT=started\nSMOKE_TEST_INIT=started again\nSMOKE_TEST_RESULT=passed",
			wantValid:           true,
			wantStageCount:      2, // only counts first occurrence
			wantMissingCount:    0,
			wantOutOfOrderCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ValidateSequence(tt.output)
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
			if len(result.Stages) != tt.wantStageCount {
				t.Errorf("Stages count = %d, want %d; stages: %v", len(result.Stages), tt.wantStageCount, result.Stages)
			}
			if len(result.MissingStages) != tt.wantMissingCount {
				t.Errorf("MissingStages count = %d, want %d; missing: %v", len(result.MissingStages), tt.wantMissingCount, result.MissingStages)
			}
			if len(result.OutOfOrderStages) != tt.wantOutOfOrderCount {
				t.Errorf("OutOfOrderStages count = %d, want %d; out of order: %v", len(result.OutOfOrderStages), tt.wantOutOfOrderCount, result.OutOfOrderStages)
			}
		})
	}
}

func TestOutputParser_ValidateSequence_LineNumbers(t *testing.T) {
	config := DefaultConfig()
	parser := NewOutputParser(config)

	output := "line 1\nSMOKE_TEST_INIT=started\nline 3\nSMOKE_TEST_RESULT=passed\nline 5"
	result := parser.ValidateSequence(output)

	if len(result.Stages) != 2 {
		t.Fatalf("Expected 2 stages, got %d", len(result.Stages))
	}

	if result.Stages[0].Name != "init" || result.Stages[0].LineNumber != 2 {
		t.Errorf("First stage: got name=%q line=%d, want name=init line=2", result.Stages[0].Name, result.Stages[0].LineNumber)
	}

	if result.Stages[1].Name != "passed" || result.Stages[1].LineNumber != 4 {
		t.Errorf("Second stage: got name=%q line=%d, want name=passed line=4", result.Stages[1].Name, result.Stages[1].LineNumber)
	}
}
