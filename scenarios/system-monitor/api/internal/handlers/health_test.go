package handlers

import "testing"

func TestParseHealthDependencyURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "operator configured http", value: "http://nodered:1880"},
		{name: "operator configured https path", value: "https://ollama.example.test/api"},
		{name: "reject unsupported scheme", value: "file:///etc/passwd", wantErr: true},
		{name: "reject missing host", value: "http:///missing-host", wantErr: true},
		{name: "reject credentials", value: "http://user:secret@example.test", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseHealthDependencyURL(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHealthDependencyURL(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
