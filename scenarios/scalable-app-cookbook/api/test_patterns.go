//go:build testing
// +build testing

package main

import (
	"net/http"
)

// PaginationScenario describes one supported pagination boundary case.
type PaginationScenario struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
}

// PaginationTestScenarios creates test cases for pagination boundaries.
func PaginationTestScenarios(basePath string) []PaginationScenario {
	return []PaginationScenario{
		{
			Name: "Pagination_ValidLimitOffset",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "10", "offset": "0"},
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "Pagination_LargeLimit",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "1000"},
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "Pagination_NegativeOffset",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"offset": "-1"},
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "Pagination_InvalidLimit",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "abc"},
			},
			ExpectedStatus: http.StatusOK,
		},
	}
}
