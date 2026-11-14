package httputil

import "testing"

func TestValidateServiceBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "http service", input: "http://127.0.0.1:15413///", want: "http://127.0.0.1:15413"},
		{name: "https service", input: "https://service.internal/api", want: "https://service.internal/api"},
		{name: "missing host", input: "http:///api", wantErr: true},
		{name: "unsupported scheme", input: "file:///tmp/service", wantErr: true},
		{name: "userinfo rejected", input: "http://user:pass@example.test", wantErr: true},
		{name: "query rejected", input: "http://example.test/?redirect=1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateServiceBaseURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got URL %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("URL = %q, want %q", got, tt.want)
			}
		})
	}
}
