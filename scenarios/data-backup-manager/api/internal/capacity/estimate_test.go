package capacity

import "testing"

func TestFirstCopyAdmitsTheCopyNotTheRetentionSum(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes: 100 << 30,
		RetainedSnapshots:  46,
		FirstCopy:          true,
		ReserveBytes:       20 << 30,
	})
	if est.Confidence != Estimated {
		t.Fatalf("confidence = %q, want estimated without a measured increment", est.Confidence)
	}
	if est.AdmissionBytes != 120<<30 {
		t.Fatalf("admission = %d, want first copy + reserve (120 GiB)", est.AdmissionBytes)
	}
	if est.ForecastBytes <= est.AdmissionBytes {
		t.Fatalf("forecast = %d, want growth beyond the first copy", est.ForecastBytes)
	}
	decision := est.Decide(157 << 30)
	if !decision.Allowed || !decision.Warning {
		t.Fatalf("decision = %+v, want admitted with a retention warning", decision)
	}
}

func TestStrictPolicyRequiresTheRetentionForecast(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes: 100 << 30,
		RetainedSnapshots:  46,
		FirstCopy:          true,
		ReserveBytes:       20 << 30,
		Policy:             PolicyStrict,
	})
	decision := est.Decide(157 << 30)
	if decision.Allowed {
		t.Fatalf("strict decision = %+v, want blocked when free space is below the forecast", decision)
	}
}

func TestIncrementalAdmissionUsesMeasuredGrowth(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes:       220 << 30,
		RetainedSnapshots:        2,
		MeasuredIncrementalBytes: 2 << 30,
		ReserveBytes:             20 << 30,
	})
	if est.AdmissionBytes != 22<<30 {
		t.Fatalf("admission = %d, want incremental + reserve (22 GiB)", est.AdmissionBytes)
	}
	decision := est.Decide(157 << 30)
	if !decision.Allowed || decision.Warning {
		t.Fatalf("decision = %+v, want admitted without warning", decision)
	}
}

func TestEstimatedDedupLowersFirstCopy(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes: 100 << 30,
		RetainedSnapshots:  2,
		FirstCopy:          true,
		MeasuredDedupRatio: 5,
		ReserveBytes:       20 << 30,
	})
	if est.FirstCopyBytes != 20<<30 {
		t.Fatalf("first copy = %d, want 20 GiB at 5x dedup", est.FirstCopyBytes)
	}
	if est.AdmissionBytes != 40<<30 {
		t.Fatalf("admission = %d, want first copy + reserve", est.AdmissionBytes)
	}
}

func TestUnknownSourceEstimateBlocksAdmission(t *testing.T) {
	est := EstimateRequired(Input{LogicalSourceBytes: 0, RetainedSnapshots: 2})
	if est.Confidence != Unknown {
		t.Fatalf("confidence = %q, want unknown", est.Confidence)
	}
	if decision := est.Decide(1 << 40); decision.Allowed {
		t.Fatalf("decision = %+v, want blocked for an unknown source estimate", decision)
	}
}

func TestAdmissionSaturatesInsteadOfOverflowing(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes:       1<<63 - 1,
		RetainedSnapshots:        1000,
		FirstCopy:                true,
		MeasuredIncrementalBytes: 1<<63 - 1,
		ReserveBytes:             1 << 30,
	})
	if est.AdmissionBytes < 0 || est.ForecastBytes < 0 {
		t.Fatalf("estimate overflowed: %+v", est)
	}
	if est.ForecastBytes != 1<<63-1 {
		t.Fatalf("forecast = %d, want saturated max int64", est.ForecastBytes)
	}
}

func TestDefaultPolicyIsAdvisory(t *testing.T) {
	est := EstimateRequired(Input{LogicalSourceBytes: 1 << 30, RetainedSnapshots: 2})
	if est.Policy != PolicyAdvisory {
		t.Fatalf("policy = %q, want advisory", est.Policy)
	}
}

func TestUnboundedRetentionAdmitsWithAnHonestWarning(t *testing.T) {
	est := EstimateRequired(Input{
		LogicalSourceBytes: 1 << 30,
		RetainedSnapshots:  0,
	})
	if !est.UnboundedRetention {
		t.Fatal("retention should be marked unbounded")
	}
	decision := est.Decide(100 << 30)
	if !decision.Allowed || !decision.Warning {
		t.Fatalf("decision = %+v, want admitted with an unbounded-retention warning", decision)
	}
	strict := est
	strict.Policy = PolicyStrict
	if strict.Decide(100 << 30).Allowed {
		t.Fatal("strict policy must not admit an unbounded retention forecast")
	}
}
