package scenario

import (
	"testing"

	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/platform-go/desktopowner"
	internal "vrooli-bridge/internal/scenario"
)

func TestDesktopAdmissionUsesExactAuthenticatedOwnerRoute(t *testing.T) {
	request := internal.Request{
		Actor: "owner", NodeID: "host", Scenario: desktopowner.Scenario,
		Service: desktopowner.Service, Method: "ReadFrame", HTTPPath: desktopowner.Service + "/ReadFrame",
	}
	node := internal.TargetNode{ID: "host"}
	if err := admit(scopecatalog.Catalog{}, nil, request, node); err != nil {
		t.Fatalf("owner frame route: %v", err)
	}
	mutations := map[string]func(*internal.Request, *internal.TargetNode){
		"agent actor":  func(r *internal.Request, _ *internal.TargetNode) { r.Actor = "agent" },
		"revoked node": func(_ *internal.Request, n *internal.TargetNode) { n.Revoked = true },
		"wrong node":   func(_ *internal.Request, n *internal.TargetNode) { n.ID = "other" },
		"wrong path":   func(r *internal.Request, _ *internal.TargetNode) { r.HTTPPath = desktopowner.Service + "/Input" },
		"signaling": func(r *internal.Request, _ *internal.TargetNode) {
			r.Method, r.HTTPPath = "Signal", desktopowner.Service+"/Signal"
		},
	}
	for name, apply := range mutations {
		t.Run(name, func(t *testing.T) {
			r, n := request, node
			apply(&r, &n)
			if err := admit(scopecatalog.Catalog{}, nil, r, n); err == nil {
				t.Fatal("unauthorized desktop route admitted")
			}
		})
	}
}
