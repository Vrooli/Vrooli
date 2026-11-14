package control

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	devicedomain "device-control/internal/devices"
)

// RelationConfirmation is the outcome of an owner-confirmed transport relation.
type RelationConfirmation struct {
	CandidateID string `json:"candidate_id"`
	CanonicalID string `json:"canonical_id"`
	MemberID    string `json:"member_id"`
	Claim       string `json:"claim"`
	Device      Device `json:"device"`
	Confirmed   bool   `json:"confirmed"`
}

// ConfirmRelation turns one same-host correlation candidate into a durable
// identity merge. The candidate alone (same host, different transports) is
// diagnostic evidence; confirmation is the owner's assertion and is refused when
// both identities report a model name and the names disagree, unless force is
// set. The identity that is not an ADB endpoint stays canonical, so the id the
// owner already knows keeps working and gains the ADB transport profile.
func (s *Service) ConfirmRelation(ctx context.Context, candidateID string, force bool) (RelationConfirmation, error) {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return RelationConfirmation{}, fmt.Errorf("candidate id is required")
	}
	var candidate *devicedomain.CorrelationCandidate
	for _, item := range s.CorrelationCandidates(ctx) {
		if item.ID == candidateID {
			copied := item
			candidate = &copied
			break
		}
	}
	if candidate == nil {
		return RelationConfirmation{}, fmt.Errorf("relation candidate %q is not currently observed; run `device-control device relations`", candidateID)
	}
	left, leftOK := s.devices.Get(candidate.LeftID)
	right, rightOK := s.devices.Get(candidate.RightID)
	if !leftOK || !rightOK {
		return RelationConfirmation{}, fmt.Errorf("both identities of candidate %q must still be retained", candidateID)
	}
	canonical, member := left, right
	if left.StrategyID == "android-adb" && right.StrategyID != "android-adb" {
		canonical, member = right, left
	}
	if !force && !modelsAgree(canonical.Model, canonical.Name, member.Model, member.Name) {
		return RelationConfirmation{}, fmt.Errorf("identities report different models (%q and %q); confirm only after checking they are one physical device, then repeat with force", firstNonEmpty(canonical.Model, canonical.Name), firstNonEmpty(member.Model, member.Name))
	}
	if len(member.Claims) == 0 {
		return RelationConfirmation{}, fmt.Errorf("identity %q has no hardware claim to assert", member.ID)
	}
	claim := fmt.Sprintf("%s=%s", member.Claims[0].Kind, member.Claims[0].Value)
	device, err := s.MergeDevices(ctx, canonical.ID, member.ID, claim)
	if err != nil {
		return RelationConfirmation{}, err
	}
	return RelationConfirmation{CandidateID: candidateID, CanonicalID: canonical.ID, MemberID: member.ID, Claim: claim, Device: device, Confirmed: true}, nil
}

// modelsAgree is true when either side lacks a model name or the normalized
// names (letters and digits only, lowercase) match or contain one another.
func modelsAgree(aModel, aName, bModel, bName string) bool {
	a, b := normalizeModel(firstNonEmpty(aModel, aName)), normalizeModel(firstNonEmpty(bModel, bName))
	if a == "" || b == "" {
		return true
	}
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}

func normalizeModel(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// AttachNetworkTransport folds a freshly onboarded network ADB identity into the
// device the owner named. The owner's naming of the target is the assertion;
// the same-host candidate and model agreement are still required.
func (s *Service) AttachNetworkTransport(ctx context.Context, adbID, targetID string) (RelationConfirmation, error) {
	targetID = strings.TrimSpace(targetID)
	if _, ok := s.devices.Get(targetID); !ok {
		return RelationConfirmation{}, fmt.Errorf("unknown device %q to attach to", targetID)
	}
	for _, candidate := range s.CorrelationCandidates(ctx) {
		if (candidate.LeftID == adbID && candidate.RightID == targetID) || (candidate.LeftID == targetID && candidate.RightID == adbID) {
			return s.ConfirmRelation(ctx, candidate.ID, false)
		}
	}
	return RelationConfirmation{}, fmt.Errorf("%q and %q were not observed on the same host, so they are not related", adbID, targetID)
}
