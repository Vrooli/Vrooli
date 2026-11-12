package resources

import (
	"context"
	"errors"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestNativeOpenRGBRetirementRequiresAuthoritativeRelease(t *testing.T) {
	_, _, ref, _, _ := nativeTransportFixture(t)
	unavailable := errors.New("native containment observation denied")
	for _, tc := range []struct {
		name   string
		absent bool
		err    error
		ok     bool
	}{{"present", false, nil, false}, {"unknown", false, unavailable, false}, {"released", true, nil, true}} {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyOpenRGBNativeRelease(t.Context(), ref, func(_ context.Context, actual platform.ServiceRef, inc string) (bool, error) {
				if !actual.Equal(ref) || inc != ref.Native.RootIncarnation {
					t.Fatal("release lost exact native incarnation")
				}
				return tc.absent, tc.err
			})
			if (err == nil) != tc.ok || tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("native retirement = %v", err)
			}
		})
	}
	uncertain := ref
	n := *ref.Native
	uncertain.Native = &n
	uncertain.RootPID = 0
	uncertain.RootStartTime = ""
	n.RootIncarnation = ""
	n.RootUniqueID = ""
	if err := verifyOpenRGBNativeRelease(t.Context(), uncertain, func(_ context.Context, _ platform.ServiceRef, inc string) (bool, error) {
		if inc != "" {
			t.Fatal("uncertain submit invented a root incarnation")
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}
