package handlers

import (
	"net/url"
	"testing"
)

func TestParsePagination_DefaultValues(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}
	query := url.Values{}

	p := ParsePagination(query, cfg)

	if p.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", p.Offset)
	}
}

func TestParsePagination_ValidValues(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}
	query := url.Values{
		"limit":  []string{"50"},
		"offset": []string{"10"},
	}

	p := ParsePagination(query, cfg)

	if p.Limit != 50 {
		t.Errorf("expected limit 50, got %d", p.Limit)
	}
	if p.Offset != 10 {
		t.Errorf("expected offset 10, got %d", p.Offset)
	}
}

func TestParsePagination_ExceedsMaxLimit(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}
	query := url.Values{
		"limit": []string{"500"},
	}

	p := ParsePagination(query, cfg)

	if p.Limit != 100 {
		t.Errorf("expected limit capped to 100, got %d", p.Limit)
	}
}

func TestParsePagination_InvalidLimit(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}

	tests := []struct {
		name  string
		value string
	}{
		{"not a number", "abc"},
		{"negative", "-5"},
		{"zero", "0"},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{}
			if tc.value != "" {
				query.Set("limit", tc.value)
			}

			p := ParsePagination(query, cfg)

			if p.Limit != 20 {
				t.Errorf("expected default limit 20, got %d", p.Limit)
			}
		})
	}
}

func TestParsePagination_InvalidOffset(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}

	tests := []struct {
		name  string
		value string
	}{
		{"not a number", "abc"},
		{"negative", "-5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{
				"offset": []string{tc.value},
			}

			p := ParsePagination(query, cfg)

			if p.Offset != 0 {
				t.Errorf("expected default offset 0, got %d", p.Offset)
			}
		})
	}
}

func TestParsePagination_ZeroOffset(t *testing.T) {
	cfg := PaginationConfig{DefaultLimit: 20, MaxLimit: 100}
	query := url.Values{
		"offset": []string{"0"},
	}

	p := ParsePagination(query, cfg)

	if p.Offset != 0 {
		t.Errorf("expected offset 0, got %d", p.Offset)
	}
}

func TestParsePagination_DifferentConfigs(t *testing.T) {
	tests := []struct {
		name           string
		cfg            PaginationConfig
		limitQuery     string
		expectedLimit  int
	}{
		{"small max", PaginationConfig{DefaultLimit: 10, MaxLimit: 25}, "50", 25},
		{"large max", PaginationConfig{DefaultLimit: 50, MaxLimit: 500}, "100", 100},
		{"uses default", PaginationConfig{DefaultLimit: 30, MaxLimit: 100}, "", 30},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{}
			if tc.limitQuery != "" {
				query.Set("limit", tc.limitQuery)
			}

			p := ParsePagination(query, tc.cfg)

			if p.Limit != tc.expectedLimit {
				t.Errorf("expected limit %d, got %d", tc.expectedLimit, p.Limit)
			}
		})
	}
}
