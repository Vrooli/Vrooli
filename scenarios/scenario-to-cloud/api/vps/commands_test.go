package vps

import "testing"

func TestValidateActionConfirmation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		action       string
		level        int
		confirmation string
		ok           bool
	}{
		{"reboot", 0, "REBOOT", true},
		{"reboot", 0, "reboot", false},
		{"stop_vrooli", 0, "my-deploy", true},
		{"stop_vrooli", 0, "other", false},
		{"cleanup", 1, "my-deploy", true},
		{"cleanup", 3, "DOCKER-RESET", true},
		{"cleanup", 3, "my-deploy", false},
		{"cleanup", 4, "RESET", true},
		{"cleanup", 5, "DELETE-VROOLI", true},
		{"cleanup", 5, "my-deploy", false},
	}
	for _, c := range cases {
		err := ValidateActionConfirmation(c.action, c.level, c.confirmation, "my-deploy")
		if (err == nil) != c.ok {
			t.Errorf("%s level %d %q: ok=%v err=%v", c.action, c.level, c.confirmation, c.ok, err)
		}
	}
}
