package identity

import "testing"

func TestClaimsAcceptOnlyHardwareKinds(t *testing.T) {
	valid := []IdentityClaim{
		{Kind: ADBSerial, Value: "R9TT608Q6MH"},
		{Kind: BluetoothMAC, Value: "E0:D8:C4:C1:2D:B1"},
		{Kind: CastID, Value: "3bc013548481e743aa769f04a9a9ba0b"},
	}
	for _, claim := range valid {
		if err := ValidateClaim(claim); err != nil {
			t.Fatalf("valid claim rejected: %#v: %v", claim, err)
		}
	}
	invalid := []IdentityClaim{
		{Kind: ClaimKind("friendly-name"), Value: "Living Room"},
		{Kind: CastID, Value: "192.168.1.158"},
		{Kind: ADBSerial, Value: "living-room.local"},
		{Kind: BluetoothMAC, Value: "not-a-mac"},
	}
	for _, claim := range invalid {
		if err := ValidateClaim(claim); err == nil {
			t.Fatalf("invalid claim accepted: %#v", claim)
		}
	}
}

func TestClaimsMatchRequiresSameKindAndValue(t *testing.T) {
	left := []IdentityClaim{{Kind: CastID, Value: "cast-1"}}
	if !ClaimsMatch(left, []IdentityClaim{{Kind: CastID, Value: "CAST-1"}}) {
		t.Fatal("same cast id should match case-insensitively")
	}
	if ClaimsMatch(left, []IdentityClaim{{Kind: ADBSerial, Value: "cast-1"}}) {
		t.Fatal("different claim kinds must not match")
	}
}
