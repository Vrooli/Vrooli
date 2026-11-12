package resources

import (
	"errors"
	platform "github.com/vrooli/platform-go"
	"testing"
)

func TestOpenRGBNativeRuntimeRejectsUnobservedOrElevatedAuthority(t *testing.T) {
	for _, tc := range []struct {
		name, os, arch string
		security       platform.ProcessSecurityContext
		err            error
		ok             bool
	}{
		{name: "darwin user", os: "darwin", arch: "arm64", security: platform.ProcessSecurityContext{Principal: "501", EffectiveUID: 501}, ok: true},
		{name: "windows user", os: "windows", arch: "amd64", security: platform.ProcessSecurityContext{Principal: "S-1-5-21-1-2-3-1001", EffectiveUID: -1}, ok: true},
		{name: "darwin root", os: "darwin", arch: "amd64", security: platform.ProcessSecurityContext{Principal: "0", Elevated: true}},
		{name: "elevated user", os: "windows", arch: "amd64", security: platform.ProcessSecurityContext{Principal: "S-1-5-21-1-2-3-1001", Elevated: true}},
		{name: "system", os: "windows", arch: "amd64", security: platform.ProcessSecurityContext{Principal: "S-1-5-18"}},
		{name: "unobserved", os: "darwin", arch: "arm64", err: errors.New("access denied")},
		{name: "empty identity", os: "windows", arch: "amd64"},
		{name: "no native asset", os: "windows", arch: "arm64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			err := validateOpenRGBNativeRuntimeAuthority(tc.os, tc.arch, func() (platform.ProcessSecurityContext, error) { called = true; return tc.security, tc.err })
			if (err == nil) != tc.ok {
				t.Fatalf("authority acceptance=%v error=%v", err == nil, err)
			}
			if tc.arch == "arm64" && tc.os == "windows" && called {
				t.Fatal("unsupported profile performed security probing")
			}
		})
	}
}
