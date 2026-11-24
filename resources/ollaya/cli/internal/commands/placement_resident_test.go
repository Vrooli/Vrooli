package commands

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// The daemon lists a digest-pinned model as `name:tag@sha256:<digest>`. A
// resident listed that way is the model, so it must never be asked of the
// broker again: its memory is already counted, and a refusal here would turn
// away a model that is already serving.
func TestResidentModelListedWithItsPinnedDigestIsRecognised(t *testing.T) {
	p := shippedModelPolicy(t)
	coba := policy.Resolution{Role: "decide.local.coba-1b", Locality: policy.LocalityLocal, Model: "coba:1b"}
	digest := p.Models["coba:1b"].Digest
	if digest == "" {
		t.Fatal("coba:1b must be digest-pinned in the shipped policy")
	}
	other := "0000000000000000000000000000000000000000000000000000000000000000"
	asked := 0
	newHandlers := func(resident string) *Handlers {
		return &Handlers{
			Residents: func(context.Context) ([]upstream.ResidentModel, error) {
				return []upstream.ResidentModel{{Name: resident, Device: "cuda:0", SizeVRAM: 4 << 30}}, nil
			},
			DaemonDevice: func(context.Context) (string, error) { return "cuda", nil },
			CapacityAdmit: func(context.Context, string, int64) (bool, string, error) {
				asked++
				return false, "insufficient vram: free memory is below the footprint", nil
			},
			LoadSlot: func(context.Context) (func(), error) { return func() {}, nil },
		}
	}

	t.Run("pinned digest of the policy: resident, broker not asked, no load slot held", func(t *testing.T) {
		asked = 0
		release, err := newHandlers("coba:1b@sha256:"+digest).requireModelPlacement(context.Background(), p, coba)
		if err != nil {
			t.Fatalf("a resident model was refused: %v", err)
		}
		release()
		if asked != 0 {
			t.Fatalf("broker asked %d times for a resident model", asked)
		}
	})
	t.Run("resolve-time check agrees", func(t *testing.T) {
		asked = 0
		if err := newHandlers("coba:1b@sha256:"+digest).checkModelPlacement(context.Background(), p, coba); err != nil {
			t.Fatalf("a resident model was refused at resolve: %v", err)
		}
		if asked != 0 {
			t.Fatalf("broker asked %d times for a resident model", asked)
		}
	})
	t.Run("a different digest of the same name is another version, not the target", func(t *testing.T) {
		asked = 0
		if err := newHandlers("coba:1b@sha256:"+other).checkModelPlacement(context.Background(), p, coba); err == nil {
			t.Fatal("a resident of another digest must not stand in for the pinned model")
		}
		if asked != 1 {
			t.Fatalf("broker asked %d times, want 1", asked)
		}
	})
}

func TestResidentIsTargetMatchesNamesTagsAndPinnedDigests(t *testing.T) {
	targets := []residentTarget{{"laya", "aaa"}, {"laya:en", "bbb"}, {"decima", ""}}
	cases := []struct {
		resident string
		want     bool
	}{
		{"laya:en", true},
		{"laya:en@sha256:bbb", true},
		{"laya:en@bbb", true},
		{"laya:en@sha256:ccc", false},
		{"laya:multilingual", false},
		{"decima:latest@sha256:anything", true}, // no pinned digest to compare against
		{"decima:latest", true},
		{"other:1b@sha256:bbb", false},
	}
	for _, tc := range cases {
		if got := residentIsTarget(tc.resident, targets); got != tc.want {
			t.Errorf("residentIsTarget(%q) = %v, want %v", tc.resident, got, tc.want)
		}
	}
}
