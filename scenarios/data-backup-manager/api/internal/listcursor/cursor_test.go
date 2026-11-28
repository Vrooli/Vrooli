package listcursor

import "testing"

func TestDecodeScopesAndBoundsOffsets(t *testing.T) {
	token := Encode("targets:alpha", 25)
	if got, err := Decode("targets:alpha", token); err != nil || got != 25 {
		t.Fatalf("decode got=%d err=%v", got, err)
	}
	if _, err := Decode("targets:beta", token); err == nil {
		t.Fatal("accepted cursor for another owner")
	}
	if _, err := Decode("plans", "bad-token"); err == nil {
		t.Fatal("accepted malformed cursor")
	}
}
