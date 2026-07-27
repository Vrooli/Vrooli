package pipeline

import (
	"scenario-to-desktop-api/build"
	"testing"
)

func TestIsArtifactReady(t *testing.T) {
	tests := []struct {
		name     string
		result   *build.PlatformResult
		expected bool
	}{
		{
			name:     "nil result returns false",
			result:   nil,
			expected: false,
		},
		{
			name:     "status ready with artifact returns true",
			result:   &build.PlatformResult{Status: BuildStatusReady, Artifact: "/path/to/artifact"},
			expected: true,
		},
		{
			name:     "status ready without artifact returns false",
			result:   &build.PlatformResult{Status: BuildStatusReady, Artifact: ""},
			expected: false,
		},
		{
			name:     "status failed returns false",
			result:   &build.PlatformResult{Status: BuildStatusFailed, Artifact: "/path/to/artifact"},
			expected: false,
		},
		{
			name:     "status building returns false",
			result:   &build.PlatformResult{Status: BuildStatusBuilding, Artifact: "/path/to/artifact"},
			expected: false,
		},
		{
			name:     "status partial returns false",
			result:   &build.PlatformResult{Status: BuildStatusPartial, Artifact: "/path/to/artifact"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsArtifactReady(tt.result)
			if result != tt.expected {
				t.Errorf("IsArtifactReady() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetReadyArtifacts(t *testing.T) {
	tests := []struct {
		name            string
		platformResults map[string]*build.PlatformResult
		expectedCount   int
		expectedKeys    []string
	}{
		{
			name:            "nil map returns empty map",
			platformResults: nil,
			expectedCount:   0,
			expectedKeys:    nil,
		},
		{
			name:            "empty map returns empty map",
			platformResults: map[string]*build.PlatformResult{},
			expectedCount:   0,
			expectedKeys:    nil,
		},
		{
			name: "all ready returns all",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
				"mac":   {Status: BuildStatusReady, Artifact: "/path/mac"},
			},
			expectedCount: 2,
			expectedKeys:  []string{"linux", "mac"},
		},
		{
			name: "mixed results returns only ready",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
				"mac":   {Status: BuildStatusFailed, Artifact: "/path/mac"},
				"win":   {Status: BuildStatusReady, Artifact: "/path/win"},
			},
			expectedCount: 2,
			expectedKeys:  []string{"linux", "win"},
		},
		{
			name: "none ready returns empty",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusFailed, Artifact: "/path/linux"},
				"mac":   {Status: BuildStatusBuilding, Artifact: ""},
			},
			expectedCount: 0,
			expectedKeys:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReadyArtifacts(tt.platformResults)
			if len(result) != tt.expectedCount {
				t.Errorf("GetReadyArtifacts() returned %d artifacts, want %d", len(result), tt.expectedCount)
			}
			for _, key := range tt.expectedKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("GetReadyArtifacts() missing expected key %s", key)
				}
			}
		})
	}
}

func TestCountReadyArtifacts(t *testing.T) {
	tests := []struct {
		name            string
		platformResults map[string]*build.PlatformResult
		expected        int
	}{
		{
			name:            "nil map returns 0",
			platformResults: nil,
			expected:        0,
		},
		{
			name: "mixed results returns ready count",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
				"mac":   {Status: BuildStatusFailed, Artifact: "/path/mac"},
				"win":   {Status: BuildStatusReady, Artifact: "/path/win"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountReadyArtifacts(tt.platformResults)
			if result != tt.expected {
				t.Errorf("CountReadyArtifacts() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestFindFirstReadyArtifact(t *testing.T) {
	tests := []struct {
		name             string
		platformResults  map[string]*build.PlatformResult
		expectEmpty      bool
		possiblePlatform []string // Since map order is random, we check for any valid value
	}{
		{
			name:            "nil map returns empty",
			platformResults: nil,
			expectEmpty:     true,
		},
		{
			name:            "empty map returns empty",
			platformResults: map[string]*build.PlatformResult{},
			expectEmpty:     true,
		},
		{
			name: "no ready returns empty",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusFailed, Artifact: "/path/linux"},
			},
			expectEmpty: true,
		},
		{
			name: "single ready returns it",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
			},
			expectEmpty:      false,
			possiblePlatform: []string{"linux"},
		},
		{
			name: "multiple ready returns one",
			platformResults: map[string]*build.PlatformResult{
				"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
				"mac":   {Status: BuildStatusReady, Artifact: "/path/mac"},
			},
			expectEmpty:      false,
			possiblePlatform: []string{"linux", "mac"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, artifact := FindFirstReadyArtifact(tt.platformResults)
			if tt.expectEmpty {
				if platform != "" || artifact != "" {
					t.Errorf("FindFirstReadyArtifact() = (%q, %q), want empty", platform, artifact)
				}
			} else {
				found := false
				for _, p := range tt.possiblePlatform {
					if platform == p {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FindFirstReadyArtifact() platform = %q, want one of %v", platform, tt.possiblePlatform)
				}
				if artifact == "" {
					t.Error("FindFirstReadyArtifact() artifact is empty, want non-empty")
				}
			}
		})
	}
}

func TestFindArtifactForPlatform(t *testing.T) {
	platformResults := map[string]*build.PlatformResult{
		"linux": {Status: BuildStatusReady, Artifact: "/path/linux"},
		"mac":   {Status: BuildStatusFailed, Artifact: "/path/mac"},
		"win":   {Status: BuildStatusReady, Artifact: ""},
	}

	tests := []struct {
		name           string
		targetPlatform string
		expected       string
	}{
		{
			name:           "existing ready platform returns artifact",
			targetPlatform: "linux",
			expected:       "/path/linux",
		},
		{
			name:           "failed platform returns empty",
			targetPlatform: "mac",
			expected:       "",
		},
		{
			name:           "ready but no artifact returns empty",
			targetPlatform: "win",
			expected:       "",
		},
		{
			name:           "non-existent platform returns empty",
			targetPlatform: "android",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindArtifactForPlatform(platformResults, tt.targetPlatform)
			if result != tt.expected {
				t.Errorf("FindArtifactForPlatform() = %q, want %q", result, tt.expected)
			}
		})
	}
}
