package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper to override validateFunc and restore afterwards
func overrideValidate(t *testing.T, fn func(*http.Request) error) func() {
	t.Helper()
	original := validateFunc
	validateFunc = fn
	return func() {
		validateFunc = original
	}
}

func overrideGetData(t *testing.T, fn func() (interface{}, error)) func() {
	t.Helper()
	original := getDataFunc
	getDataFunc = fn
	return func() {
		getDataFunc = original
	}
}

func overrideValidateErr(t *testing.T, fn func(error) error) func() {
	t.Helper()
	original := validateErrFunc
	validateErrFunc = fn
	return func() {
		validateErrFunc = original
	}
}

func TestHandleMultipleStatusesErrorBranches(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cases := []struct {
		name           string
		override       func(*http.Request) error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "ValidationError",
			override:       func(*http.Request) error { return errors.New("validation") },
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation",
		},
		{
			name:           "InternalError",
			override:       func(*http.Request) error { return errors.New("boom") },
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "boom",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			restore := overrideValidate(t, tc.override)
			defer restore()

			suite := HandlerTestSuite{
				HandlerName: "handleMultipleStatuses",
				Handler:     handleMultipleStatuses,
				BaseURL:     "/statuses",
			}

			patterns := NewTestScenarioBuilder().
				AddCustom(ErrorTestPattern{
					Name:           tc.name,
					ExpectedStatus: tc.expectedStatus,
					Execute: func(t *testing.T, _ interface{}) *HTTPTestRequest {
						return &HTTPTestRequest{Method: http.MethodPost, Path: "/statuses"}
					},
					Validate: func(t *testing.T, w *httptest.ResponseRecorder, _ interface{}) {
						assertJSONResponse(t, w, tc.expectedStatus, map[string]interface{}{"error": tc.expectedError})
					},
				}).
				Build()

			suite.RunErrorTests(t, patterns)
		})
	}
}

func TestHandleDifferentErrorNameReturnsJSONWithoutStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	restore := overrideValidate(t, func(*http.Request) error { return errors.New("naming issue") })
	defer restore()

	suite := HandlerTestSuite{
		HandlerName: "handleDifferentErrorName",
		Handler:     handleDifferentErrorName,
		BaseURL:     "/diff-error",
	}

	patterns := NewTestScenarioBuilder().
		AddCustom(ErrorTestPattern{
			Name:           "ReturnsBody",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, _ interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodGet, Path: "/diff-error"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, _ interface{}) {
				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"error": "naming issue"})
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}

func TestHandleNestedErrorsValidationPath(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	restoreData := overrideGetData(t, func() (interface{}, error) { return nil, errors.New("fetch failed") })
	defer restoreData()

	restoreValidation := overrideValidateErr(t, func(err error) error { return errors.New("invalid data") })
	defer restoreValidation()

	suite := HandlerTestSuite{
		HandlerName: "handleNestedErrors",
		Handler:     handleNestedErrors,
		BaseURL:     "/nested",
	}

	patterns := NewTestScenarioBuilder().
		AddCustom(ErrorTestPattern{
			Name:           "ValidationFailure",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, _ interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodGet, Path: "/nested"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, _ interface{}) {
				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"error": "invalid data"})
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}

func TestHandleNestedErrorsSwallowsErrorWhenValidationNil(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	restoreData := overrideGetData(t, func() (interface{}, error) { return nil, errors.New("fetch failed") })
	defer restoreData()

	restoreValidation := overrideValidateErr(t, func(err error) error { return nil })
	defer restoreValidation()

	suite := HandlerTestSuite{
		HandlerName: "handleNestedErrors",
		Handler:     handleNestedErrors,
		BaseURL:     "/nested",
	}

	patterns := NewTestScenarioBuilder().
		AddCustom(ErrorTestPattern{
			Name:           "ValidationAllowsPassThrough",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, _ interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodGet, Path: "/nested"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, _ interface{}) {
				if w.Body.Len() != 0 {
					t.Fatalf("expected empty body when validation returns nil, got %s", w.Body.String())
				}
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}
