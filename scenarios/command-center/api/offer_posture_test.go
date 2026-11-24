package main

import "testing"

func TestOfferPostureAcceptsCompleteConnectShapeAndRejectsPartialOrAliasedData(t *testing.T) {
	position := map[string]any{
		"cashMinor": "82000", "burnMinor": "4000", "revenueMinor": "12000",
		"runwayMonths": float64(20.5), "runwayAvailable": true, "currency": "USD",
	}
	root := map[string]any{
		"position": position, "postureSource": "money-ledger.position: operator-book",
		"postureAgeSeconds": "95", "defaultAliveGap": "default-alive threshold met with 1.4 buffer",
	}
	selected, ok := offerPosture(root)
	if !ok {
		t.Fatal("complete typed Connect response rejected")
	}
	if selected["cashMinor"] != float64(82000) || selected["currency"] != "USD" || selected["ageSeconds"] != float64(95) {
		t.Fatalf("selected=%+v, want original money unit and producer age", selected)
	}

	for name, mutate := range map[string]func(map[string]any){
		"partial position": func(payload map[string]any) { payload["position"].(map[string]any)["partial"] = true },
		"snake-case alias": func(payload map[string]any) {
			delete(payload["position"].(map[string]any), "cashMinor")
			payload["position"].(map[string]any)["cash_minor"] = float64(82000)
		},
		"missing currency":       func(payload map[string]any) { delete(payload["position"].(map[string]any), "currency") },
		"malformed producer age": func(payload map[string]any) { payload["postureAgeSeconds"] = "old" },
	} {
		t.Run(name, func(t *testing.T) {
			payload := map[string]any{
				"position": map[string]any{
					"cashMinor": float64(82000), "burnMinor": float64(4000), "revenueMinor": float64(12000),
					"runwayMonths": float64(20.5), "runwayAvailable": true, "currency": "USD",
				},
				"postureSource": "money-ledger.position", "postureAgeSeconds": float64(95),
			}
			mutate(payload)
			if _, ok := offerPosture(payload); ok {
				t.Fatal("invalid/partial Connect response accepted")
			}
		})
	}
}
