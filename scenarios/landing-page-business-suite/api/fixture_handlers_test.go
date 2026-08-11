package main

import (
	"net/http/httptest"
	"testing"
)

func TestFixtureRequestAllowedOnlyOnLoopbackDevelopmentAuthority(t *testing.T) {
	tests := []struct {
		name string
		host string
		env  string
		want bool
	}{
		{name: "localhost", host: "localhost:1234", want: true},
		{name: "ipv4 loopback", host: "127.0.0.1:1234", want: true},
		{name: "ipv6 loopback", host: "[::1]:1234", want: true},
		{name: "remote host", host: "lpbs.example.test", want: false},
		{name: "production", host: "localhost:1234", env: "production", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LPBS_ENVIRONMENT", test.env)
			r := httptest.NewRequest("POST", "http://"+test.host+"/api/v1/dev/fixtures/seed", nil)
			if got := fixtureRequestAllowed(r); got != test.want {
				t.Fatalf("fixtureRequestAllowed() = %v, want %v", got, test.want)
			}
		})
	}
}
