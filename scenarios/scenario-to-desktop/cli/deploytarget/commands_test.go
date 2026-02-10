package deploytarget

import (
	"errors"
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
