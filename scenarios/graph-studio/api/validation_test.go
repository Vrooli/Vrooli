package main

import (
	"fmt"
	"testing"
)

// TestValidateGraphName tests graph name validation
func TestValidateGraphName(t *testing.T) {
	gv := NewGraphValidator(make(map[string]*Plugin))

	// Create a valid long name (255 chars)
	longValidName := "My Graph"
	for len(longValidName) < 255 {
		longValidName += " Test"
	}
	longValidName = longValidName[:255]

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid name", "My Test Graph", false},
		{"empty name", "", true},
		{"name too long", "A" + string(make([]byte, 300)), true},
		{"valid min length", "ABC", false},
		{"too short", "AB", true},
		{"valid max length", longValidName, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gv.validateGraphName(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateGraphName(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

// TestValidateGraphType tests graph type validation
func TestValidateGraphType(t *testing.T) {
	// Setup validator with test plugins
	plugins := map[string]*Plugin{
		"mind-maps": {ID: "mind-maps", Name: "Mind Maps", Enabled: true},
		"bpmn":      {ID: "bpmn", Name: "BPMN", Enabled: true},
		"disabled":  {ID: "disabled", Name: "Disabled Plugin", Enabled: false},
	}
	gv := NewGraphValidator(plugins)

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid type - mind-maps", "mind-maps", false},
		{"valid type - bpmn", "bpmn", false},
		{"empty type", "", true},
		{"unknown type", "unknown-plugin", true},
		{"disabled plugin", "disabled", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gv.validateGraphType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateGraphType(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

// TestValidateDescription tests description validation
func TestValidateDescription(t *testing.T) {
	gv := NewGraphValidator(make(map[string]*Plugin))

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"empty description", "", false},
		{"valid description", "This is a test graph", false},
		{"description too long", string(make([]byte, 2001)), true},
		{"max valid length", string(make([]byte, 2000)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gv.validateDescription(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateDescription() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateTags tests tag validation
func TestValidateTags(t *testing.T) {
	gv := NewGraphValidator(make(map[string]*Plugin))

	// Create valid tags for tests
	validTags20 := make([]string, 20)
	for i := 0; i < 20; i++ {
		validTags20[i] = fmt.Sprintf("tag%d", i)
	}

	tooManyTags := make([]string, 21)
	for i := 0; i < 21; i++ {
		tooManyTags[i] = fmt.Sprintf("tag%d", i)
	}

	tests := []struct {
		name      string
		input     []string
		wantError bool
	}{
		{"nil tags", nil, false},
		{"empty tags", []string{}, false},
		{"valid tags", []string{"tag1", "tag2", "tag3"}, false},
		{"too many tags", tooManyTags, true},
		{"max valid tags", validTags20, false},
		{"tag too long", []string{string(make([]byte, 101))}, true},
		{"empty tag", []string{""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := gv.validateTags(tt.input)
			if (len(errs) > 0) != tt.wantError {
				t.Errorf("validateTags() errors = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}

// TestValidateMetadata tests metadata validation
func TestValidateMetadata(t *testing.T) {
	gv := NewGraphValidator(make(map[string]*Plugin))

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{"nil metadata", nil, false},
		{"empty metadata", map[string]interface{}{}, false},
		{"valid metadata", map[string]interface{}{"key": "value"}, false},
		{"nested metadata", map[string]interface{}{"nested": map[string]interface{}{"key": "value"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gv.validateMetadata(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateMetadata() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateCreateGraphRequest tests full request validation
func TestValidateCreateGraphRequest(t *testing.T) {
	plugins := map[string]*Plugin{
		"mind-maps": {ID: "mind-maps", Name: "Mind Maps", Enabled: true},
	}
	gv := NewGraphValidator(plugins)

	tests := []struct {
		name      string
		request   *CreateGraphRequest
		wantError bool
	}{
		{
			name: "valid request",
			request: &CreateGraphRequest{
				Name:        "Test Graph",
				Type:        "mind-maps",
				Description: "A test graph",
				Tags:        []string{"test"},
			},
			wantError: false,
		},
		{
			name: "missing name",
			request: &CreateGraphRequest{
				Type:        "mind-maps",
				Description: "Test",
			},
			wantError: true,
		},
		{
			name: "invalid type",
			request: &CreateGraphRequest{
				Name: "Test",
				Type: "invalid-type",
			},
			wantError: true,
		},
		{
			name: "name too long",
			request: &CreateGraphRequest{
				Name: string(make([]byte, 300)),
				Type: "mind-maps",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := gv.ValidateCreateGraphRequest(tt.request)
			if (len(errs) > 0) != tt.wantError {
				t.Errorf("ValidateCreateGraphRequest() errors = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}

// TestValidateUpdateGraphRequest tests update request validation
func TestValidateUpdateGraphRequest(t *testing.T) {
	plugins := map[string]*Plugin{
		"mind-maps": {ID: "mind-maps", Name: "Mind Maps", Enabled: true},
	}
	gv := NewGraphValidator(plugins)

	tests := []struct {
		name      string
		request   *UpdateGraphRequest
		wantError bool
	}{
		{
			name: "valid update",
			request: &UpdateGraphRequest{
				Name:        "Updated Name",
				Description: "Updated description",
			},
			wantError: false,
		},
		{
			name:      "empty update",
			request:   &UpdateGraphRequest{},
			wantError: false,
		},
		{
			name: "name too long",
			request: &UpdateGraphRequest{
				Name: string(make([]byte, 300)),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := gv.ValidateUpdateGraphRequest(tt.request)
			if (len(errs) > 0) != tt.wantError {
				t.Errorf("ValidateUpdateGraphRequest() errors = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}
