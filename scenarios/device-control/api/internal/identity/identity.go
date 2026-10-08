// Package identity owns the evidence-bearing identity claims used to
// reconcile observations from independent device transports.
package identity

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type ClaimKind string

const (
	ADBSerial    ClaimKind = "adb-serial"
	BluetoothMAC ClaimKind = "bluetooth-mac"
	CastID       ClaimKind = "cast-id"
)

var bluetoothMAC = regexp.MustCompile(`(?i)^[0-9a-f]{2}([:-][0-9a-f]{2}){5}$`)

// IdentityClaim is the smallest durable explanation for why two transport
// observations may refer to one physical device. Evidence is intentionally
// explicit so an owner assertion cannot be mistaken for discovery evidence.
type IdentityClaim struct {
	Kind       ClaimKind `json:"kind"`
	Value      string    `json:"value"`
	StrategyID string    `json:"strategy_id"`
	Evidence   string    `json:"evidence"`
}

func (c IdentityClaim) Key() string {
	return string(c.Kind) + "\x00" + strings.ToLower(strings.TrimSpace(c.Value))
}

func (c IdentityClaim) Valid() bool { return ValidateClaim(c) == nil }

func ValidateClaim(claim IdentityClaim) error {
	claim.Value = strings.TrimSpace(claim.Value)
	if claim.Value == "" {
		return fmt.Errorf("identity claim value is required")
	}
	switch claim.Kind {
	case ADBSerial, CastID:
		if isNetworkIdentity(claim.Value) {
			return fmt.Errorf("%s cannot be an address, hostname, or mDNS instance name", claim.Kind)
		}
	case BluetoothMAC:
		if !bluetoothMAC.MatchString(claim.Value) {
			return fmt.Errorf("bluetooth-mac claim must be a MAC address")
		}
	default:
		return fmt.Errorf("identity claim kind %q is not accepted", claim.Kind)
	}
	return nil
}

func NewClaim(kind, value, strategyID, evidence string) (IdentityClaim, error) {
	claim := IdentityClaim{Kind: ClaimKind(strings.TrimSpace(kind)), Value: strings.TrimSpace(value), StrategyID: strings.TrimSpace(strategyID), Evidence: strings.TrimSpace(evidence)}
	if err := ValidateClaim(claim); err != nil {
		return IdentityClaim{}, err
	}
	if claim.Evidence == "" {
		claim.Evidence = "observed"
	}
	return claim, nil
}

func ClaimsMatch(left, right []IdentityClaim) bool {
	for _, a := range left {
		for _, b := range right {
			if a.Kind == b.Kind && strings.EqualFold(strings.TrimSpace(a.Value), strings.TrimSpace(b.Value)) {
				return true
			}
		}
	}
	return false
}

func ClaimFor(strategyID, kind, value, evidence string) (IdentityClaim, error) {
	return NewClaim(kind, value, strategyID, evidence)
}

func isNetworkIdentity(value string) bool {
	if net.ParseIP(value) != nil || strings.Contains(value, ".local") {
		return true
	}
	if strings.Contains(value, ":") && !bluetoothMAC.MatchString(value) {
		return true
	}
	return strings.ContainsAny(value, " /\\")
}
