// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// [REQ:REQ-P0-005] CLI Tool Expectation Config - HTTP handler tests
// [REQ:REQ-P0-004] Structural Expectation Config - HTTP handler tests
package handlers

import (
	"bytes"
	"context"
	"development-toolchain-validator/domain/expectation"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// newTestExpectationHandler creates a handler with mock repositories for testing.
func newTestExpectationHandler() *ExpectationHandler {
	structRepo := newMockStructuralRepository()
	cliRepo := newMockCLIRepository()
	svc := expectation.NewService(structRepo, cliRepo)
	return NewExpectationHandler(svc)
}

// mockStructuralRepository is a test mock.
type mockStructuralRepository struct {
	expectations map[string]*expectation.StructuralExpectation
}

func newMockStructuralRepository() *mockStructuralRepository {
	return &mockStructuralRepository{
		expectations: make(map[string]*expectation.StructuralExpectation),
	}
}

func (m *mockStructuralRepository) Create(_ context.Context, input expectation.CreateStructuralInput) (*expectation.StructuralExpectation, error) {
	exp := &expectation.StructuralExpectation{
		ID:           "exp-123",
		ConnectionID: input.ConnectionID,
		Type:         input.Type,
		Pattern:      input.Pattern,
		Required:     input.Required,
	}
	m.expectations[exp.ID] = exp
	return exp, nil
}

func (m *mockStructuralRepository) GetByID(_ context.Context, id string) (*expectation.StructuralExpectation, error) {
	exp, ok := m.expectations[id]
	if !ok {
		return nil, expectation.ErrNotFound
	}
	return exp, nil
}

func (m *mockStructuralRepository) List(_ context.Context, opts expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	var result []*expectation.StructuralExpectation
	for _, exp := range m.expectations {
		if opts.ConnectionID == "" || exp.ConnectionID == opts.ConnectionID {
			result = append(result, exp)
		}
	}
	return result, nil
}

func (m *mockStructuralRepository) Delete(_ context.Context, id string) error {
	if _, ok := m.expectations[id]; !ok {
		return expectation.ErrNotFound
	}
	delete(m.expectations, id)
	return nil
}

func (m *mockStructuralRepository) DeleteByConnection(_ context.Context, connectionID string) error {
	for id, exp := range m.expectations {
		if exp.ConnectionID == connectionID {
			delete(m.expectations, id)
		}
	}
	return nil
}

// mockCLIRepository is a test mock.
type mockCLIRepository struct {
	assertions map[string]*expectation.CLIAssertion
}

func newMockCLIRepository() *mockCLIRepository {
	return &mockCLIRepository{
		assertions: make(map[string]*expectation.CLIAssertion),
	}
}

func (m *mockCLIRepository) Create(_ context.Context, input expectation.CreateCLIInput) (*expectation.CLIAssertion, error) {
	assertion := &expectation.CLIAssertion{
		ID:           "cli-123",
		ConnectionID: input.ConnectionID,
		Command:      input.Command,
		JSONPath:     input.JSONPath,
		Operator:     input.Operator,
	}
	m.assertions[assertion.ID] = assertion
	return assertion, nil
}

func (m *mockCLIRepository) GetByID(_ context.Context, id string) (*expectation.CLIAssertion, error) {
	assertion, ok := m.assertions[id]
	if !ok {
		return nil, expectation.ErrNotFound
	}
	return assertion, nil
}

func (m *mockCLIRepository) List(_ context.Context, opts expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	var result []*expectation.CLIAssertion
	for _, assertion := range m.assertions {
		if opts.ConnectionID == "" || assertion.ConnectionID == opts.ConnectionID {
			result = append(result, assertion)
		}
	}
	return result, nil
}

func (m *mockCLIRepository) Delete(_ context.Context, id string) error {
	if _, ok := m.assertions[id]; !ok {
		return expectation.ErrNotFound
	}
	delete(m.assertions, id)
	return nil
}

func (m *mockCLIRepository) DeleteByConnection(_ context.Context, connectionID string) error {
	for id, assertion := range m.assertions {
		if assertion.ConnectionID == connectionID {
			delete(m.assertions, id)
		}
	}
	return nil
}

// errorMockStructuralRepository always returns an error on List.
type errorMockStructuralRepository struct{}

func (e *errorMockStructuralRepository) Create(_ context.Context, _ expectation.CreateStructuralInput) (*expectation.StructuralExpectation, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockStructuralRepository) GetByID(_ context.Context, _ string) (*expectation.StructuralExpectation, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockStructuralRepository) List(_ context.Context, _ expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockStructuralRepository) Delete(_ context.Context, _ string) error {
	return expectation.ErrNotFound
}

func (e *errorMockStructuralRepository) DeleteByConnection(_ context.Context, _ string) error {
	return expectation.ErrNotFound
}

// errorMockCLIRepository always returns an error on List.
type errorMockCLIRepository struct{}

func (e *errorMockCLIRepository) Create(_ context.Context, _ expectation.CreateCLIInput) (*expectation.CLIAssertion, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockCLIRepository) GetByID(_ context.Context, _ string) (*expectation.CLIAssertion, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockCLIRepository) List(_ context.Context, _ expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	return nil, expectation.ErrNotFound
}

func (e *errorMockCLIRepository) Delete(_ context.Context, _ string) error {
	return expectation.ErrNotFound
}

func (e *errorMockCLIRepository) DeleteByConnection(_ context.Context, _ string) error {
	return expectation.ErrNotFound
}

// newErrorExpectationHandler creates a handler that returns errors for list operations.
func newErrorExpectationHandler() *ExpectationHandler {
	structRepo := &errorMockStructuralRepository{}
	cliRepo := &errorMockCLIRepository{}
	svc := expectation.NewService(structRepo, cliRepo)
	return NewExpectationHandler(svc)
}

// TestExpectationHandler_CreateStructural tests structural expectation creation.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_CreateStructural(t *testing.T) {
	handler := newTestExpectationHandler()

	input := expectation.CreateStructuralInput{
		ConnectionID: "conn-123",
		Type:         expectation.TypeFolder,
		Pattern:      "api/domain",
		Required:     true,
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/expectations/structural", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateStructural(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("CreateStructural() status = %d, want %d", rr.Code, http.StatusCreated)
	}
}

// TestExpectationHandler_CreateStructural_DryRun tests dry-run validation.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_CreateStructural_DryRun(t *testing.T) {
	handler := newTestExpectationHandler()

	tests := []struct {
		name       string
		input      expectation.CreateStructuralInput
		wantStatus int
	}{
		{
			name: "valid input",
			input: expectation.CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         expectation.TypeFolder,
				Pattern:      "api/domain",
				Required:     true,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid - missing connection ID",
			input: expectation.CreateStructuralInput{
				Type:    expectation.TypeFolder,
				Pattern: "api/",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/expectations/structural", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			handler.CreateStructural(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("CreateStructural() dry-run status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_CreateCLI tests CLI assertion creation.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_CreateCLI(t *testing.T) {
	handler := newTestExpectationHandler()

	input := expectation.CreateCLIInput{
		ConnectionID:  "conn-123",
		Command:       "scenario-auditor standards scan ref-123 --wait --json",
		JSONPath:      "$.security.violations",
		Operator:      expectation.OpEq,
		ExpectedValue: 0,
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/expectations/cli", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateCLI(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("CreateCLI() status = %d, want %d", rr.Code, http.StatusCreated)
	}
}

// TestExpectationHandler_CreateCLI_DryRun tests dry-run validation.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_CreateCLI_DryRun(t *testing.T) {
	handler := newTestExpectationHandler()

	tests := []struct {
		name       string
		input      expectation.CreateCLIInput
		wantStatus int
	}{
		{
			name: "valid input",
			input: expectation.CreateCLIInput{
				ConnectionID:  "conn-123",
				Command:       "scenario-auditor standards scan ref-123 --wait --json",
				JSONPath:      "$.score",
				Operator:      expectation.OpGte,
				ExpectedValue: 80,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid - dangerous command",
			input: expectation.CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "rm -rf /",
				JSONPath:     "$.result",
				Operator:     expectation.OpEq,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid - bad JSONPath",
			input: expectation.CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "scenario-auditor standards scan ref-123 --wait --json",
				JSONPath:     "invalid",
				Operator:     expectation.OpEq,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/expectations/cli", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			handler.CreateCLI(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("CreateCLI() dry-run status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_ListStructural tests listing structural expectations.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_ListStructural(t *testing.T) {
	handler := newTestExpectationHandler()

	req := httptest.NewRequest(http.MethodGet, "/expectations/structural", nil)
	rr := httptest.NewRecorder()

	handler.ListStructural(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListStructural() status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestExpectationHandler_ListCLI tests listing CLI assertions.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_ListCLI(t *testing.T) {
	handler := newTestExpectationHandler()

	req := httptest.NewRequest(http.MethodGet, "/expectations/cli", nil)
	rr := httptest.NewRecorder()

	handler.ListCLI(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListCLI() status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestExpectationHandler_ListStructural_Error tests error handling for listing structural expectations.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_ListStructural_Error(t *testing.T) {
	handler := newErrorExpectationHandler()

	req := httptest.NewRequest(http.MethodGet, "/expectations/structural", nil)
	rr := httptest.NewRecorder()

	handler.ListStructural(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ListStructural() error case status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

// TestExpectationHandler_ListCLI_Error tests error handling for listing CLI assertions.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_ListCLI_Error(t *testing.T) {
	handler := newErrorExpectationHandler()

	req := httptest.NewRequest(http.MethodGet, "/expectations/cli", nil)
	rr := httptest.NewRecorder()

	handler.ListCLI(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ListCLI() error case status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

// TestExpectationHandler_GetStructural tests getting a structural expectation by ID.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_GetStructural(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockStructuralRepository)
		id         string
		wantStatus int
		category   string
	}{
		{
			name: "existing_expectation",
			setupMock: func(m *mockStructuralRepository) {
				m.expectations["exp-123"] = &expectation.StructuralExpectation{
					ID:           "exp-123",
					ConnectionID: "conn-123",
					Type:         expectation.TypeFolder,
					Pattern:      "api/domain",
					Required:     true,
				}
			},
			id:         "exp-123",
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "nonexistent_expectation",
			setupMock:  func(m *mockStructuralRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			tc.setupMock(structRepo)
			cliRepo := newMockCLIRepository()
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/expectations/structural/"+tc.id, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("GetStructural() status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_GetCLI tests getting a CLI assertion by ID.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_GetCLI(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockCLIRepository)
		id         string
		wantStatus int
		category   string
	}{
		{
			name: "existing_assertion",
			setupMock: func(m *mockCLIRepository) {
				m.assertions["cli-123"] = &expectation.CLIAssertion{
					ID:           "cli-123",
					ConnectionID: "conn-123",
					Command:      "scenario-auditor standards scan ref-123 --wait --json",
					JSONPath:     "$.security.violations",
					Operator:     expectation.OpEq,
				}
			},
			id:         "cli-123",
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "nonexistent_assertion",
			setupMock:  func(m *mockCLIRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			cliRepo := newMockCLIRepository()
			tc.setupMock(cliRepo)
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/expectations/cli/"+tc.id, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("GetCLI() status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_DeleteStructural tests deleting a structural expectation.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_DeleteStructural(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockStructuralRepository)
		id         string
		wantStatus int
		category   string
	}{
		{
			name: "delete_existing",
			setupMock: func(m *mockStructuralRepository) {
				m.expectations["exp-123"] = &expectation.StructuralExpectation{
					ID:           "exp-123",
					ConnectionID: "conn-123",
					Type:         expectation.TypeFolder,
					Pattern:      "api/domain",
					Required:     true,
				}
			},
			id:         "exp-123",
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "delete_nonexistent",
			setupMock:  func(m *mockStructuralRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			tc.setupMock(structRepo)
			cliRepo := newMockCLIRepository()
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/expectations/structural/"+tc.id, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("DeleteStructural() status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_DeleteCLI tests deleting a CLI assertion.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_DeleteCLI(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockCLIRepository)
		id         string
		wantStatus int
		category   string
	}{
		{
			name: "delete_existing",
			setupMock: func(m *mockCLIRepository) {
				m.assertions["cli-123"] = &expectation.CLIAssertion{
					ID:           "cli-123",
					ConnectionID: "conn-123",
					Command:      "scenario-auditor standards scan ref-123 --wait --json",
					JSONPath:     "$.security.violations",
					Operator:     expectation.OpEq,
				}
			},
			id:         "cli-123",
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "delete_nonexistent",
			setupMock:  func(m *mockCLIRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			cliRepo := newMockCLIRepository()
			tc.setupMock(cliRepo)
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/expectations/cli/"+tc.id, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("DeleteCLI() status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_CreateStructural_InvalidJSON tests error handling for invalid JSON.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_CreateStructural_InvalidJSON(t *testing.T) {
	handler := newTestExpectationHandler()

	req := httptest.NewRequest(http.MethodPost, "/expectations/structural", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateStructural(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateStructural() with invalid JSON status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	// Verify error response content
	var response struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if response.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// TestExpectationHandler_CreateCLI_InvalidJSON tests error handling for invalid JSON.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_CreateCLI_InvalidJSON(t *testing.T) {
	handler := newTestExpectationHandler()

	req := httptest.NewRequest(http.MethodPost, "/expectations/cli", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateCLI(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateCLI() with invalid JSON status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	// Verify error response content
	var response struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if response.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// TestExpectationHandler_DeleteStructural_DryRun tests dry-run mode for delete structural.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_DeleteStructural_DryRun(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockStructuralRepository)
		id         string
		wantStatus int
	}{
		{
			name: "existing_expectation",
			setupMock: func(m *mockStructuralRepository) {
				m.expectations["exp-123"] = &expectation.StructuralExpectation{
					ID:           "exp-123",
					ConnectionID: "conn-123",
					Type:         expectation.TypeFolder,
					Pattern:      "api/domain",
					Required:     true,
				}
			},
			id:         "exp-123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent_expectation",
			setupMock:  func(m *mockStructuralRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			tc.setupMock(structRepo)
			cliRepo := newMockCLIRepository()
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/expectations/structural/"+tc.id, nil)
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("DeleteStructural() dry-run status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_DeleteCLI_DryRun tests dry-run mode for delete CLI.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_DeleteCLI_DryRun(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockCLIRepository)
		id         string
		wantStatus int
	}{
		{
			name: "existing_assertion",
			setupMock: func(m *mockCLIRepository) {
				m.assertions["cli-123"] = &expectation.CLIAssertion{
					ID:           "cli-123",
					ConnectionID: "conn-123",
					Command:      "scenario-auditor standards scan ref-123 --wait --json",
					JSONPath:     "$.security.violations",
					Operator:     expectation.OpEq,
				}
			},
			id:         "cli-123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent_assertion",
			setupMock:  func(m *mockCLIRepository) {},
			id:         "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structRepo := newMockStructuralRepository()
			cliRepo := newMockCLIRepository()
			tc.setupMock(cliRepo)
			svc := expectation.NewService(structRepo, cliRepo)
			handler := NewExpectationHandler(svc)

			router := mux.NewRouter()
			handler.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/expectations/cli/"+tc.id, nil)
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("DeleteCLI() dry-run status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestExpectationHandler_ListCLI_WithConnectionFilter tests filtering CLI assertions by connection ID.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_ListCLI_WithConnectionFilter(t *testing.T) {
	handler := newTestExpectationHandler()

	// Create assertions for different connections
	for _, connID := range []string{"conn-123", "conn-456"} {
		input := expectation.CreateCLIInput{
			ConnectionID:  connID,
			Command:       "scenario-auditor standards scan ref --wait --json",
			JSONPath:      "$.score",
			Operator:      expectation.OpGte,
			ExpectedValue: 80,
		}
		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/expectations/cli", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.CreateCLI(rr, req)
	}

	// Test filtering by connection_id
	req := httptest.NewRequest(http.MethodGet, "/expectations/cli?connection_id=conn-123", nil)
	rr := httptest.NewRecorder()

	handler.ListCLI(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListCLI() with filter status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Verify response structure
	var response struct {
		Assertions []interface{} `json:"assertions"`
		Count      int           `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// TestExpectationHandler_CreateStructural_ValidationErrors tests all validation error types.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_CreateStructural_ValidationErrors(t *testing.T) {
	handler := newTestExpectationHandler()

	tests := []struct {
		name           string
		input          expectation.CreateStructuralInput
		wantStatus     int
		wantMsgContain string
		category       string
	}{
		{
			name: "invalid_type",
			input: expectation.CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         expectation.ExpectationType("invalid_type"),
				Pattern:      "api/domain",
			},
			wantStatus:     http.StatusBadRequest,
			wantMsgContain: "Invalid expectation type",
			category:       "error",
		},
		{
			name: "empty_pattern",
			input: expectation.CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         expectation.TypeFile,
				Pattern:      "",
			},
			wantStatus:     http.StatusBadRequest,
			wantMsgContain: "Invalid pattern",
			category:       "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/expectations/structural", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			handler.CreateStructural(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("CreateStructural() status = %d, want %d, body: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}

			if tc.wantMsgContain != "" && !bytes.Contains(rr.Body.Bytes(), []byte(tc.wantMsgContain)) {
				t.Errorf("CreateStructural() body should contain %q, got: %s", tc.wantMsgContain, rr.Body.String())
			}
		})
	}
}

// TestExpectationHandler_CreateCLI_ValidationErrors tests all CLI validation error types.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
func TestExpectationHandler_CreateCLI_ValidationErrors(t *testing.T) {
	handler := newTestExpectationHandler()

	tests := []struct {
		name           string
		input          expectation.CreateCLIInput
		wantStatus     int
		wantMsgContain string
		category       string
	}{
		{
			name: "invalid_operator",
			input: expectation.CreateCLIInput{
				ConnectionID:  "conn-123",
				Command:       "echo test",
				JSONPath:      "$.result",
				Operator:      expectation.AssertionOperator("invalid_op"),
				ExpectedValue: 0,
			},
			wantStatus:     http.StatusBadRequest,
			wantMsgContain: "Invalid assertion operator",
			category:       "error",
		},
		{
			name: "empty_command",
			input: expectation.CreateCLIInput{
				ConnectionID:  "conn-123",
				Command:       "",
				JSONPath:      "$.result",
				Operator:      expectation.OpEq,
				ExpectedValue: 0,
			},
			wantStatus:     http.StatusBadRequest,
			wantMsgContain: "Invalid CLI command",
			category:       "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/expectations/cli", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dry-Run", "true")
			rr := httptest.NewRecorder()

			handler.CreateCLI(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("CreateCLI() status = %d, want %d, body: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}

			if tc.wantMsgContain != "" && !bytes.Contains(rr.Body.Bytes(), []byte(tc.wantMsgContain)) {
				t.Errorf("CreateCLI() body should contain %q, got: %s", tc.wantMsgContain, rr.Body.String())
			}
		})
	}
}

// TestExpectationHandler_ListStructural_WithConnectionFilter tests filtering by connection ID.
// [REQ:REQ-P0-004] Structural Expectation Config
func TestExpectationHandler_ListStructural_WithConnectionFilter(t *testing.T) {
	handler := newTestExpectationHandler()

	// Create expectations for different connections
	for _, connID := range []string{"conn-123", "conn-456"} {
		input := expectation.CreateStructuralInput{
			ConnectionID: connID,
			Type:         expectation.TypeFolder,
			Pattern:      "api/domain",
			Required:     true,
		}
		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/expectations/structural", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.CreateStructural(rr, req)
	}

	// Test filtering by connection_id
	req := httptest.NewRequest(http.MethodGet, "/expectations/structural?connection_id=conn-123", nil)
	rr := httptest.NewRecorder()

	handler.ListStructural(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListStructural() with filter status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Verify response structure
	var response struct {
		Expectations []interface{} `json:"expectations"`
		Count        int           `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}
