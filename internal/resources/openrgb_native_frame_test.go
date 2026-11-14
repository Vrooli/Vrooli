package resources

import (
	"encoding/json"
	"testing"
)

func TestNativeOpenRGBFrameFieldsPreserveTaggedDefaultsAndMapData(t *testing.T) {
	type mapped struct {
		Values map[string]string `json:"values"`
	}
	if err := validateOpenRGBNativeFrameFields([]byte(`{"values":{"CaseSensitive":"first","casesensitive":"second"}}`), new(mapped)); err != nil {
		t.Fatal("data map keys were mistaken for struct authority fields", err)
	}
	_, m, ref, w, _ := nativeTransportFixture(t)
	for _, frame := range []openRGBNativeClaimFrame{{Operation: "prepared", Ref: &ref, Material: &m, OK: true}, {Operation: "claim", Ref: &ref, Witness: &w}, {Operation: "claimed", OK: true}} {
		data, err := json.Marshal(frame)
		if err != nil {
			t.Fatal(err)
		}
		if err = validateOpenRGBNativeFrameFields(data, new(openRGBNativeClaimFrame)); err != nil {
			t.Fatalf("canonical production frame rejected: %v", err)
		}
	}
}
