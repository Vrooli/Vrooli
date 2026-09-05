package runsignal

// CrossScenarioMetrics are generic measures over commands that opted into the
// manifest semantics contract. No scenario or command name is encoded here.
type CrossScenarioMetrics struct {
	Available        bool   `json:"available"`
	Reason           string `json:"reason,omitempty"`
	DeclaredCalls    int    `json:"declaredCalls"`
	ClassifiedBase   int    `json:"classifiedBase"`
	QueryRefinement  int    `json:"queryRefinement"`
	QueryAbandonment int    `json:"queryAbandonment"`
	VerifyCycle      int    `json:"verifyCycle"`
	VerifyRegression int    `json:"verifyRegression"`
	GuidanceAdoption int    `json:"guidanceAdoption"`
}

func CrossScenario(facts []InvocationFact) CrossScenarioMetrics {
	result := CrossScenarioMetrics{}
	families := map[string][]InvocationFact{}
	for _, fact := range facts {
		if fact.SemanticsKind == "" {
			continue
		}
		result.DeclaredCalls++
		family := fact.Executable + " " + fact.CommandPath
		families[family] = append(families[family], fact)
	}
	result.ClassifiedBase = result.DeclaredCalls
	if result.DeclaredCalls == 0 {
		result.Reason = "no invocation command declares cross-scenario semantics"
		return result
	}
	result.Available = true
	for _, calls := range families {
		for i := 1; i < len(calls); i++ {
			if calls[i].SemanticsKind == "query" && calls[i-1].SemanticsKind == "query" && calls[i].IntentClass != calls[i-1].IntentClass {
				result.QueryRefinement++
			}
		}
		for i, call := range calls {
			switch call.SemanticsKind {
			case "query":
				if i == len(calls)-1 && call.Outcome != "success" {
					result.QueryAbandonment++
				}
			case "verify":
				if call.Outcome != "success" {
					result.VerifyCycle++
				}
			}
		}
		seenPass := false
		for _, call := range calls {
			if call.SemanticsKind == "verify" && call.Outcome == "success" {
				seenPass = true
			}
			if call.SemanticsKind == "verify" && seenPass && call.Outcome == "failure" {
				result.VerifyRegression++
			}
		}
	}
	for i, fact := range facts {
		if fact.SemanticsKind != "guidance" {
			continue
		}
		before, after := map[string]bool{}, map[string]bool{}
		for _, candidate := range facts[:i] {
			if candidate.SemanticsKind != "guidance" {
				before[candidate.Executable+" "+candidate.CommandPath] = true
			}
		}
		for _, candidate := range facts[i+1:] {
			if candidate.SemanticsKind != "guidance" {
				after[candidate.Executable+" "+candidate.CommandPath] = true
			}
		}
		for family := range after {
			if !before[family] {
				result.GuidanceAdoption++
				break
			}
		}
	}
	return result
}
