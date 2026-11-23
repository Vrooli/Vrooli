package ops

import "testing"

func TestValidateParallaxRequestRequiresPreparedInputs(t *testing.T) {
	const want = "ops: prepared parallax parameters are required"

	tests := []struct {
		name   string
		params *Params
	}{
		{name: "nil params"},
		{name: "nil parallax", params: &Params{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateParallaxRequest(tt.params); err == nil || err.Error() != want {
				t.Fatalf("ValidateParallaxRequest() error = %v, want exactly %q", err, want)
			}
		})
	}
}
