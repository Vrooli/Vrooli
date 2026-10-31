package goalhome

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
)

// JourneyReceipt is an owner observation, never a count copied from prose.
// Applicability and execution success remain separate for each exact case.
type JourneyReceipt struct {
	ID          string
	Scenario    string
	CompletedAt time.Time
	Cases       []JourneyCaseEvidence
}

type JourneyCaseEvidence struct {
	Path    string
	Passed  bool
	Current bool
	Reason  string
}

type JourneyReceiptReader func(string) (JourneyReceipt, error)

// ResolveJourneyReceipts replaces only explicitly receipt-backed journey
// results. Review/custom gates and independent acceptance remain untouched.
func ResolveJourneyReceipts(report *EpochReport, scenario string, read JourneyReceiptReader) {
	cache := map[string]JourneyReceipt{}
	errors := map[string]error{}
	for i := range report.Gates {
		gate := &report.Gates[i]
		if gate.Result != "receipt" || gate.State == GateDropped {
			continue
		}
		gate.State = GateUnmet
		if gate.Type != GateJourney {
			gate.Detail = "owner receipts are supported only for journey gates"
			continue
		}
		cases, err := receiptGateCases(gate.Spec)
		if err != nil {
			gate.Detail = err.Error()
			continue
		}
		refs := strings.Split(gate.Value, ",")
		if len(refs) == 0 || len(refs) > 32 {
			gate.Detail = "receipt gate requires 1–32 exact owner run references"
			continue
		}
		selected := map[string]struct {
			at       time.Time
			evidence JourneyCaseEvidence
		}{}
		failed := ""
		for _, ref := range refs {
			ref = strings.TrimSpace(ref)
			id, ok := strings.CutPrefix(ref, "workflow-health:")
			if !ok || id == "" || strings.ContainsAny(id, "/\\ \t\n") {
				failed = "invalid workflow-health receipt reference"
				break
			}
			receipt, known := cache[id]
			if !known && errors[id] == nil {
				receipt, err = read(id)
				if err == nil {
					cache[id] = receipt
				} else {
					errors[id] = err
				}
			}
			if errors[id] != nil {
				failed = "owner receipt unavailable: " + errors[id].Error()
				break
			}
			if receipt.ID != id || receipt.Scenario != scenario || receipt.CompletedAt.IsZero() {
				failed = "receipt identity, scenario or terminal timestamp does not match this goal"
				break
			}
			for _, item := range receipt.Cases {
				prior, exists := selected[item.Path]
				if !exists || receipt.CompletedAt.After(prior.at) || (receipt.CompletedAt.Equal(prior.at) && (!item.Passed || !item.Current)) {
					selected[item.Path] = struct {
						at       time.Time
						evidence JourneyCaseEvidence
					}{receipt.CompletedAt, item}
				}
			}
		}
		if failed != "" {
			gate.Detail = failed
			continue
		}
		var missing []string
		for _, name := range cases {
			item, ok := selected[name]
			if !ok {
				missing = append(missing, name+": no owner evidence")
				continue
			}
			if !item.evidence.Passed || !item.evidence.Current {
				missing = append(missing, name+": "+item.evidence.Reason)
			}
		}
		if len(missing) != 0 {
			gate.Detail = strings.Join(missing, "; ")
			continue
		}
		gate.State = GateMet
		gate.Detail = fmt.Sprintf("%d/%d declared cases passed with current owner evidence", len(cases), len(cases))
		gate.EvidenceRefs = slices.Clone(refs)
	}
	if report.GateStatus != GatesLegacy {
		report.GateStatus = GateMet
		for _, gate := range report.Gates {
			if gate.State == GateUnmet {
				report.GateStatus = GateUnmet
			}
		}
	}
}

func receiptGateCases(spec string) ([]string, error) {
	list, owner, ok := strings.Cut(spec, "@")
	if !ok || strings.TrimSpace(owner) != "workflow-health" {
		return nil, fmt.Errorf("receipt journey gate needs exact case paths @ workflow-health")
	}
	var cases []string
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, "bas/cases/") || path.Clean(name) != name || !strings.HasSuffix(name, ".json") || strings.ContainsAny(name, "*?[]\\ \t\n") {
			return nil, fmt.Errorf("receipt journey gate requires exact registered bas/cases/*.json paths")
		}
		if slices.Contains(cases, name) {
			return nil, fmt.Errorf("duplicate required workflow case %s", name)
		}
		cases = append(cases, name)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("receipt journey gate has no required cases")
	}
	return cases, nil
}
