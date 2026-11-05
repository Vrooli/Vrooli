package gates

import "testing"

func differentialObservationForTest(context string, x float64, duration string) differentialObservation {
	item := differentialObservation{
		AssetID:     "controls.test",
		ComponentID: "test",
		ExampleName: "default",
		StateID:     "default",
		ViewportID:  "desktop",
		ViewportW:   100,
		AX:          axObservation{Bounds: &axBounds{X: x, Width: 20}, ComputedStyle: map[string]string{"transitionDuration": duration}},
	}
	if context == "ltr" || context == "rtl" {
		item.Direction = context
		item.Locale = map[string]string{"ltr": "en", "rtl": "ar"}[context]
	} else {
		item.Motion = context
	}
	return item
}

func TestRTLIsARealDifferentialClaim(t *testing.T) {
	pass, message := evaluateRTL([]differentialObservation{
		differentialObservationForTest("ltr", 10, "0s"),
		differentialObservationForTest("rtl", 70, "0s"),
	})
	if pass != "pass" || message != "" {
		t.Fatalf("mirrored captures = %q/%q, want pass", pass, message)
	}
	fail, message := evaluateRTL([]differentialObservation{
		differentialObservationForTest("ltr", 10, "0s"),
		differentialObservationForTest("rtl", 10, "0s"),
	})
	if fail != "renders-not-differential" || message == "" {
		t.Fatalf("hard-coded captures = %q/%q, want named differential failure", fail, message)
	}
}

func TestReducedMotionRequiresRenderedDurationChange(t *testing.T) {
	pass, _ := evaluateReducedMotion([]differentialObservation{
		differentialObservationForTest("no-preference", 10, "200ms"),
		differentialObservationForTest("reduce", 10, "0ms"),
	})
	if pass != "pass" {
		t.Fatalf("reduced capture = %q, want pass", pass)
	}
	fail, message := evaluateReducedMotion([]differentialObservation{
		differentialObservationForTest("no-preference", 10, "200ms"),
		differentialObservationForTest("reduce", 10, "200ms"),
	})
	if fail != "renders-not-differential" || message == "" {
		t.Fatalf("hard-coded motion = %q/%q, want named differential failure", fail, message)
	}
}

func TestDifferentialWithoutBothContextsIsUnmeasured(t *testing.T) {
	verdict, message := evaluateDifferential("rtl", []differentialObservation{differentialObservationForTest("ltr", 10, "0s")})
	if verdict != "unmeasured" || message != "" {
		t.Fatalf("single context = %q/%q, want unmeasured", verdict, message)
	}
}
