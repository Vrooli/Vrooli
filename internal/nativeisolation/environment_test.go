package nativeisolation

import (
	"strings"
	"testing"
)

func TestFiniteEnvironmentNarrowing(t *testing.T) {
	m, r := fixture()
	rt, e := NewRuntime(m)
	if e != nil {
		t.Fatal(e)
	}
	env, e := rt.PrepareEnvironment(r.Repository, []string{"HOME=/home/host", "PATH=/host/bin", "UNAPPROVED_SECRET=fixture"})
	if e != nil || strings.Join(env, ";") != "HOME="+r.Repository+"/.native-home;PATH=/usr/bin:/bin" {
		t.Fatalf("narrowing: %v %v", env, e)
	}
	for _, input := range [][]string{{"HOME=a", "HOME=b"}, {"LD_PRELOAD=fixture"}, {"HTTPS_PROXY=fixture"}, {"DBUS_SESSION_BUS_ADDRESS=fixture"}, {"bad=fixture"}, {"NO_EQUALS"}} {
		if _, e := rt.PrepareEnvironment(r.Repository, input); e == nil {
			t.Fatalf("accepted %v", input)
		}
	}
	if _, e := rt.PrepareEnvironment("/wrong", nil); e == nil {
		t.Fatal("wrong work view accepted")
	}
	rt.manifest.Enabled = false
	if _, e := rt.PrepareEnvironment(r.Repository, nil); e == nil {
		t.Fatal("disabled accepted")
	}
}
