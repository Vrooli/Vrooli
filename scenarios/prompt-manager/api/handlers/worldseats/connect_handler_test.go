package worldseats

import (
	"encoding/json"
	"testing"

	worldseatsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/worldseats"
)

func TestConfigConversionIsDeterministicAndLossless(t *testing.T) {
	wire := &worldseatsv1.WorldSeats{Groups: []*worldseatsv1.SeatGroup{
		{FurnitureType: "table", Seats: []*worldseatsv1.Seat{{Position: &worldseatsv1.Position{X: 1, Y: 2, Z: 3}, Rotation: 1.5}}},
		{FurnitureType: "chair", Seats: []*worldseatsv1.Seat{{Position: &worldseatsv1.Position{X: -1, Y: 0.5, Z: 4}, Rotation: -0.25}}},
	}}
	encoded, err := json.Marshal(encodeConfig(wire))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeConfig(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Groups) != 2 || decoded.Groups[0].FurnitureType != "chair" || decoded.Groups[1].FurnitureType != "table" {
		t.Fatalf("groups were not sorted deterministically: %#v", decoded.Groups)
	}
	seat := decoded.Groups[1].Seats[0]
	if seat.Position.X != 1 || seat.Position.Y != 2 || seat.Position.Z != 3 || seat.Rotation != 1.5 {
		t.Fatalf("round-trip changed seat: %#v", seat)
	}
}
