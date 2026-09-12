package routes

import (
	"testing"

	"github.com/stretchr/testify/require"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
)

// TestPublicExposureFlag maps every accepted --public-exposure value (and the
// empty/unset case) to the proto enum, and rejects unknown values. Empty must
// map to UNSPECIFIED so create defaults to inherit and update leaves the
// existing override untouched.
func TestPublicExposureFlag(t *testing.T) {
	cases := []struct {
		in      string
		want    routesv1.PublicExposure
		wantErr bool
	}{
		{"", routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED, false},
		{"  ", routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED, false},
		{"inherit", routesv1.PublicExposure_PUBLIC_EXPOSURE_INHERIT, false},
		{"ENABLED", routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED, false},
		{"Disabled", routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED, false},
		{"bogus", routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED, true},
	}
	for _, c := range cases {
		got, err := publicExposureFlag(c.in)
		if c.wantErr {
			require.Errorf(t, err, "input %q should be rejected", c.in)
			continue
		}
		require.NoErrorf(t, err, "input %q", c.in)
		require.Equalf(t, c.want, got, "input %q", c.in)
	}
}

// TestPublicExposureLabel collapses unspecified and inherit to the same display
// label so unset routes read as the default rather than as noise.
func TestPublicExposureLabel(t *testing.T) {
	require.Equal(t, "inherit", publicExposureLabel(routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED))
	require.Equal(t, "inherit", publicExposureLabel(routesv1.PublicExposure_PUBLIC_EXPOSURE_INHERIT))
	require.Equal(t, "enabled", publicExposureLabel(routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED))
	require.Equal(t, "disabled", publicExposureLabel(routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED))
}
