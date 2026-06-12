package dependencies

import "testing"

func TestParseVersionOutput(t *testing.T) {
	tests := []struct {
		name    string
		command string
		output  string
		want    Version
	}{
		{name: "go", command: "go", output: "go version go1.25.0 linux/amd64", want: Version{Major: 1, Minor: 25}},
		{name: "node", command: "node", output: "v22.1.3", want: Version{Major: 22, Minor: 1, Patch: 3}},
		{name: "python", command: "python3", output: "Python 3.13.0", want: Version{Major: 3, Minor: 13}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionOutput(tt.command, tt.output)
			if err != nil {
				t.Fatalf("ParseVersionOutput: %v", err)
			}
			if got != tt.want {
				t.Fatalf("version = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	if !((Version{Major: 1, Minor: 25}).AtLeast(Version{Major: 1, Minor: 21})) {
		t.Fatal("expected newer minor version to satisfy minimum")
	}
	if (Version{Major: 18, Minor: 0}).AtLeast(Version{Major: 20}) {
		t.Fatal("expected older major version to fail minimum")
	}
}
