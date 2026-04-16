package deploytarget

import (
	"errors"
	"strings"
	"testing"
)

func TestIsServiceAuthReadinessError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "lpbs secret missing",
			err:  errors.New("api error (400): LPBS_SERVICE_SECRET is not set"),
			want: true,
		},
		{
			name: "service auth disabled",
			err:  errors.New("api error (400): service auth is not configured"),
			want: true,
		},
		{
			name: "other api error",
			err:  errors.New("api error (500): internal error"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isServiceAuthReadinessError(tc.err)
			if got != tc.want {
				t.Fatalf("isServiceAuthReadinessError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuildServiceAuthNextSteps(t *testing.T) {
	t.Run("scenario-to-desktop-secret-missing", func(t *testing.T) {
		err := errors.New("api error (400): LPBS_SERVICE_SECRET is not set for scenario-to-desktop runtime (checked env and ~/.vrooli/secrets.json)")
		steps := buildServiceAuthNextSteps(err, "prod")
		if !strings.Contains(steps, "--scenario scenario-to-desktop") {
			t.Fatalf("expected scenario-to-desktop guidance, got: %s", steps)
		}
		if !strings.Contains(steps, "test prod --require-service-auth") {
			t.Fatalf("expected retry command for target prod, got: %s", steps)
		}
	})

	t.Run("lpbs-runtime-auth-missing", func(t *testing.T) {
		err := errors.New("api error (400): service auth is not configured in landing-page-business-suite runtime")
		steps := buildServiceAuthNextSteps(err, "prod")
		if !strings.Contains(steps, "--scenario landing-page-business-suite") {
			t.Fatalf("expected LPBS runtime guidance, got: %s", steps)
		}
		if !strings.Contains(steps, "service-auth-status --require-enabled") {
			t.Fatalf("expected service-auth-status command, got: %s", steps)
		}
	})
}
