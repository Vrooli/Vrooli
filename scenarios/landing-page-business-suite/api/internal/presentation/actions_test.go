package presentation

import "testing"

func TestActionKeyUsesTheStableJSONTupleWithoutHTMLRewriting(t *testing.T) {
	got := ActionKey(Action{Kind: ActionOpen, AppKey: "web-console", PlanRef: "café", Target: "/go?label=<unicode>"})
	want := `["open","web-console","café","/go?label=<unicode>"]`
	if got != want {
		t.Fatalf("ActionKey() = %q, want %q", got, want)
	}
}

func TestActionKeyMatchesJavaScriptSeparatorsWithoutCorruptingLiteralEscapes(t *testing.T) {
	for _, target := range []string{"/line\u2028separator\u2029", `/literal\u2028\u2029`} {
		got := ActionKey(Action{Kind: ActionOpen, Target: target})
		want := "[\"open\",\"\",\"\",\"/line\u2028separator\u2029\"]"
		if target[1:5] == "lite" {
			want = `["open","","","/literal\\u2028\\u2029"]`
		}
		if got != want {
			t.Fatalf("ActionKey(%q)=%q want %q", target, got, want)
		}
	}
}

func TestRepeatedCTAHasOneOwnerObservationAndOpenNeedsLaunchOwner(t *testing.T) { // [REQ:LP-PRES-011]
	action := Action{Kind: ActionDownload, AppKey: "web-console"}
	result := ResolveResult{Page: ResolvedPage{
		Display: PageDisplay{Shell: ShellDisplay{UnavailableReason: "No released installer", HeaderAction: &action}},
		Blocks:  []ResolvedBlock{{Content: ProductHeroContent{Actions: []Action{action}}}, {Content: ClosingActionContent{Actions: []Action{action, {Kind: ActionOpen, AppKey: "web-console"}}}}},
	}, Spotlights: []ResolvedAppSpotlight{{AppKey: "web-console", DetailRoute: "/apps/aquila"}}}
	JoinActionOwners(&result, ActionOwnerObservations{Downloads: map[string]ActionOwnerObservation{"web-console": {Ready: true, Href: "/downloads/aquila"}}})
	if len(result.Actions) != 2 {
		t.Fatalf("duplicate wire owner identities: %#v", result.Actions)
	}
	if result.Actions[0].Status != ResolvedActionReady {
		t.Fatal("qualified download unavailable")
	}
	if result.Actions[1].Status != ResolvedActionUnavailable {
		t.Fatal("marketing app detail mistaken for a launch destination")
	}
	JoinActionOwners(&result, ActionOwnerObservations{})
	if result.Actions[0].Status != ResolvedActionUnavailable || result.Actions[0].Href != "" {
		t.Fatal("stale owner readiness survived new observation")
	}
}

func TestOwnerHrefCannotBypassSafeNavigationPolicy(t *testing.T) { // [REQ:LP-PRES-012]
	for _, href := range []string{"javascript:alert(1)", "//attacker.test", "/\\attacker.test", "/%5cattacker.test", "https://user:secret@attacker.test", "/%00"} {
		result := ResolveResult{Page: ResolvedPage{Display: PageDisplay{Shell: ShellDisplay{HeaderAction: &Action{Kind: ActionPurchase, PlanRef: "price"}}}}}
		JoinActionOwners(&result, ActionOwnerObservations{Purchases: map[string]ActionOwnerObservation{"price": {Ready: true, Href: href}}})
		if result.Actions[0].Status != ResolvedActionUnavailable {
			t.Fatalf("unsafe owner href accepted: %s", href)
		}
	}
}

func TestResolveActionsIncludesHeaderAndEveryBlockAction(t *testing.T) {
	result := ResolveResult{Page: ResolvedPage{
		Display: PageDisplay{Shell: ShellDisplay{UnavailableReason: "Configured unavailable", HeaderAction: &Action{Kind: ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"}}},
		Blocks:  []ResolvedBlock{{Kind: BlockProductHero, Content: ProductHeroContent{Actions: []Action{{Kind: ActionAnchor, Label: "Jump", AccessibleLabel: "Jump", Target: "#hero"}}}}},
	}, Spotlights: []ResolvedAppSpotlight{{AppKey: "web-console", DetailRoute: "/apps/aquila"}}}
	actions := ResolveActions(result)
	if len(actions) != 2 {
		t.Fatalf("resolved action count = %d, want 2", len(actions))
	}
	if actions[0].Status != ResolvedActionUnavailable || actions[0].Reason != "Configured unavailable" {
		t.Fatalf("owner action observation = %#v", actions[0])
	}
	if actions[1].Status != ResolvedActionReady || actions[1].Href != "#hero" {
		t.Fatalf("local action observation = %#v", actions[1])
	}
}

func TestJoinActionOwnersDoesNotDiscardReadablePage(t *testing.T) {
	result := ResolveResult{Page: ResolvedPage{
		Display: PageDisplay{Shell: ShellDisplay{UnavailableReason: "Unavailable", HeaderAction: &Action{Kind: ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"}}},
		Blocks:  []ResolvedBlock{{Kind: BlockProductHero, Content: ProductHeroContent{Actions: []Action{{Kind: ActionAnchor, Label: "Jump", AccessibleLabel: "Jump", Target: "#hero"}}}}},
	}, Spotlights: []ResolvedAppSpotlight{}}
	result.Actions = ResolveActions(result)
	JoinActionOwners(&result, ActionOwnerObservations{Purchases: map[string]ActionOwnerObservation{"price-pro": {Ready: true, Href: "/checkout?price_id=price-pro"}}})
	if result.Actions[0].Status != ResolvedActionReady || result.Actions[0].Href != "/checkout?price_id=price-pro" {
		t.Fatalf("purchase owner join = %#v", result.Actions[0])
	}
	if result.Actions[1].Status != ResolvedActionReady || result.Actions[1].Href != "#hero" {
		t.Fatalf("local action changed during owner join = %#v", result.Actions[1])
	}
}
