package validation

import "testing"

func TestSharedSecurityHeaderAdoption(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"canonical", `package main; import "github.com/vrooli/api-core/apihttp"; func main(){ serve(apihttp.SecurityHeaders(router)) }`, true},
		{"alias", `package main; import boundary "github.com/vrooli/api-core/apihttp"; func main(){ serve(boundary.SecurityHeaders(router)) }`, true},
		{"unused", `package main; import "github.com/vrooli/api-core/apihttp"; func main(){ serve(router) }`, false},
		{"untrusted", `package main; import "example.com/apihttp"; func main(){ serve(apihttp.SecurityHeaders(router)) }`, false},
		{"comment", `package main; // apihttp.SecurityHeaders(router)`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := usesSharedSecurityHeaders(tc.source); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
