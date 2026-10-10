package probeprotocol

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestFixedObservationContractIsCanonicalBoundedAndNonarbitrary(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	o := Observation{Repository: "/approved/repository", CanaryID: strings.Repeat("a", 32), WorkspaceSHA256: MarkerDigest(strings.Repeat("a", 32), "workspace"), DeniedSHA256: MarkerDigest(strings.Repeat("a", 32), "denied"), Deadline: now.Add(5 * time.Minute)}
	encoded, e := o.Encode(now)
	if e != nil {
		t.Fatal(e)
	}
	copy, e := Decode(encoded, now)
	if e != nil || copy != o {
		t.Fatal(copy, e)
	}
	if o.WorkspaceMarker() != "/approved/repository/.vrooli-isolation-canary/"+o.CanaryID+"/workspace.marker" {
		t.Fatal("arbitrary marker")
	}
	for _, change := range []func(*Observation){func(o *Observation) { o.CanaryID = "../escape" }, func(o *Observation) { o.Repository = "/" }, func(o *Observation) { o.Repository = "/approved/../foreign" }, func(o *Observation) { o.Repository = "/approved/repository:foreign" }, func(o *Observation) { o.Deadline = now }, func(o *Observation) { o.Deadline = now.Add(5*time.Minute + time.Nanosecond) }, func(o *Observation) { o.WorkspaceSHA256 = "invalid" }} {
		c := o
		change(&c)
		if _, e := c.Encode(now); e == nil {
			t.Fatal("expanded observation accepted")
		}
	}
	for _, raw := range []string{"invalid", strings.Repeat("a", 2801), base64.RawURLEncoding.EncodeToString([]byte(`{"command":"sh"}`)), encoded + "="} {
		if _, e := Decode(raw, now); e == nil {
			t.Fatal("arbitrary/noncanonical input accepted")
		}
	}
}
