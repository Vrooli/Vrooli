package worldseats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedConfig writes a Config JSON file into the temp store directory.
func seedConfig(t *testing.T, storeDir string, cfg Config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, configFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// validate
// ---------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "valid: single chair",
			cfg:     Config{"chair": {{Position: [3]float64{0, 0, 0}, Rotation: 0}}},
			wantErr: "",
		},
		{
			name:    "valid: empty config",
			cfg:     Config{},
			wantErr: "",
		},
		{
			name: "valid: all known types",
			cfg: Config{
				"chair":        {},
				"bench":        {},
				"stool":        {},
				"armchair":     {},
				"desk":         {},
				"table":        {},
				"picnic-table": {},
				"coffee-table": {},
				"campfire":     {},
			},
			wantErr: "",
		},
		{
			name: "valid: positions at boundaries",
			cfg: Config{
				"chair": {
					{Position: [3]float64{10, 10, 10}, Rotation: 0},
					{Position: [3]float64{-10, -10, -10}, Rotation: 0},
				},
			},
			wantErr: "",
		},
		{
			name: "valid: 20 seats (max)",
			cfg: func() Config {
				seats := make([]SeatPosition, 20)
				for i := range seats {
					seats[i] = SeatPosition{Position: [3]float64{0, 0, 0}}
				}
				return Config{"chair": seats}
			}(),
			wantErr: "",
		},
		{
			name:    "invalid: unknown type",
			cfg:     Config{"sofa": {{Position: [3]float64{0, 0, 0}}}},
			wantErr: "unknown furniture type: sofa",
		},
		{
			name: "invalid: 21 seats",
			cfg: func() Config {
				seats := make([]SeatPosition, 21)
				for i := range seats {
					seats[i] = SeatPosition{Position: [3]float64{0, 0, 0}}
				}
				return Config{"chair": seats}
			}(),
			wantErr: "too many seats (21 > 20)",
		},
		{
			name:    "invalid: position[0] above max",
			cfg:     Config{"chair": {{Position: [3]float64{10.1, 0, 0}}}},
			wantErr: "position[0] out of range",
		},
		{
			name:    "invalid: position[2] below min",
			cfg:     Config{"chair": {{Position: [3]float64{0, 0, -10.1}}}},
			wantErr: "position[2] out of range",
		},
		{
			name:    "invalid: position[1] above max",
			cfg:     Config{"chair": {{Position: [3]float64{0, 11, 0}}}},
			wantErr: "position[1] out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HandleGet
// ---------------------------------------------------------------------------

func TestHandleGet(t *testing.T) {
	t.Run("file_exists", func(t *testing.T) {
		dir := t.TempDir()
		cfg := Config{"chair": {{Position: [3]float64{1, 2, 3}, Rotation: 0.5}}}
		seedConfig(t, dir, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/world-seats", nil)
		rec := httptest.NewRecorder()
		HandleGet(dir)(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var got Config
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		seats := got["chair"]
		if len(seats) != 1 {
			t.Fatalf("expected 1 chair seat, got %d", len(seats))
		}
		if seats[0].Position != [3]float64{1, 2, 3} {
			t.Fatalf("unexpected position: %v", seats[0].Position)
		}
	})

	t.Run("file_missing", func(t *testing.T) {
		dir := t.TempDir()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/world-seats", nil)
		rec := httptest.NewRecorder()
		HandleGet(dir)(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var got Config
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty config, got %v", got)
		}
	})

	t.Run("corrupt_file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, configFile), []byte("{invalid json"), 0o644); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/world-seats", nil)
		rec := httptest.NewRecorder()
		HandleGet(dir)(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// HandlePut
// ---------------------------------------------------------------------------

func TestHandlePut(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"chair":[{"position":[1,0,0],"rotation":0.5}]}`

		req := httptest.NewRequest(http.MethodPut, "/api/v1/world-seats", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		HandlePut(dir)(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify file was written
		data, err := os.ReadFile(filepath.Join(dir, configFile))
		if err != nil {
			t.Fatal(err)
		}
		var got Config
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if len(got["chair"]) != 1 {
			t.Fatalf("expected 1 chair seat in file, got %d", len(got["chair"]))
		}
	})

	t.Run("invalid_json", func(t *testing.T) {
		dir := t.TempDir()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/world-seats", strings.NewReader("{bad"))
		rec := httptest.NewRecorder()
		HandlePut(dir)(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("unknown_type", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"sofa":[{"position":[0,0,0],"rotation":0}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/world-seats", strings.NewReader(body))
		rec := httptest.NewRecorder()
		HandlePut(dir)(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "unknown furniture type") {
			t.Fatalf("unexpected error: %s", rec.Body.String())
		}
	})

	t.Run("too_many_seats", func(t *testing.T) {
		dir := t.TempDir()
		seats := make([]SeatPosition, 21)
		for i := range seats {
			seats[i] = SeatPosition{Position: [3]float64{0, 0, 0}}
		}
		cfg := Config{"chair": seats}
		data, _ := json.Marshal(cfg)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/world-seats", strings.NewReader(string(data)))
		rec := httptest.NewRecorder()
		HandlePut(dir)(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "too many seats") {
			t.Fatalf("unexpected error: %s", rec.Body.String())
		}
	})

	t.Run("position_out_of_range", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"chair":[{"position":[10.1,0,0],"rotation":0}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/world-seats", strings.NewReader(body))
		rec := httptest.NewRecorder()
		HandlePut(dir)(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "out of range") {
			t.Fatalf("unexpected error: %s", rec.Body.String())
		}
	})
}
