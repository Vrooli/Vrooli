//go:build darwin

package hostreqkit

import (
	"errors"
	osuser "os/user"
	"testing"
)

func TestDarwinInvokingHomeUsesAuthoritativeUIDWithoutRootHOMEFallback(t *testing.T) {
	t.Setenv("HOME", "/var/root")
	home, err := resolveDarwinInvokingHome(501, func(uid string) (*osuser.User, error) {
		if uid != "501" {
			t.Fatal("different UID looked up")
		}
		return &osuser.User{Uid: "501", HomeDir: "/Users/operator"}, nil
	})
	if err != nil || home != "/Users/operator" {
		t.Fatalf("got %q, %v", home, err)
	}
	for _, user := range []*osuser.User{nil, {Uid: "502", HomeDir: "/Users/operator"}, {Uid: "501", HomeDir: "relative"}, {Uid: "501", HomeDir: "/"}} {
		if _, err := resolveDarwinInvokingHome(501, func(string) (*osuser.User, error) { return user, nil }); err == nil {
			t.Fatal("unqualified native home mapping accepted")
		}
	}
	lookupErr := errors.New("native directory unavailable")
	if _, err = resolveDarwinInvokingHome(501, func(string) (*osuser.User, error) { return nil, lookupErr }); !errors.Is(err, lookupErr) {
		t.Fatal("lookup failure fell back to root HOME", err)
	}
}
