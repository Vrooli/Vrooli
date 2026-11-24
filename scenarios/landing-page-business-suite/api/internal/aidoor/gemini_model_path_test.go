package aidoor

import "testing"

func TestGeminiModelFromPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "door generate content",
			path: "/api/v1/ai/models/fixture-model:generateContent",
			want: "fixture-model",
		},
		{
			name: "door stream generate content",
			path: "/api/v1/ai/models/google/gemini-3.1-pro:streamGenerateContent",
			want: "google/gemini-3.1-pro",
		},
		{
			name: "antigravity custom endpoint URI with scheme",
			path: "/api/v1/ai/v1beta/models/http:/127.0.0.1:34567/api/v1/ai/models/fixture-model:streamGenerateContent",
			want: "fixture-model",
		},
		{
			name: "antigravity custom endpoint URI with host authority",
			path: "/api/v1/ai/v1beta/models/127.0.0.1:34567/models/fixture-model:streamGenerateContent",
			want: "fixture-model",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := geminiModelFromPath(tc.path); got != tc.want {
				t.Fatalf("geminiModelFromPath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestIsGeminiGenerateContentPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{path: "/api/v1/ai/models/fixture-model:generateContent", want: true},
		{path: "/api/v1/ai/v1beta/models/127.0.0.1:34567/models/fixture-model:streamGenerateContent", want: true},
		{path: "/api/v1/ai/v1beta/models/fixture-model", want: false},
		{path: "/api/v1/ai/v1beta/models/fixture-model:other", want: false},
	} {
		if got := isGeminiGenerateContentPath(tc.path); got != tc.want {
			t.Fatalf("isGeminiGenerateContentPath(%q) = %t, want %t", tc.path, got, tc.want)
		}
	}
}
