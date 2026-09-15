package actspace

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
)

func TestServesActDenominator(t *testing.T) { // [REQ:PRT-P0-007]
	_, definition := liveActDefinition(t)
	if definition.Projection != spacedoc.ProjectionAct {
		t.Fatalf("projection=%q, want act", definition.Projection)
	}
	if definition.Owner != "program-runtime" {
		t.Fatalf("owner=%q, want program-runtime", definition.Owner)
	}
	if len(definition.Cells) != 28 {
		t.Fatalf("cells=%d, want complete Act denominator", len(definition.Cells))
	}
}

func TestMatchesSpacedocContract(t *testing.T) { // [REQ:PRT-P0-007]
	_, definition := liveActDefinition(t)
	data, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip spacedoc.SpaceDefinition
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Projection != spacedoc.ProjectionAct || roundTrip.Owner != "program-runtime" || roundTrip.DenominatorConfidence == spacedoc.ConfidenceSketch {
		t.Fatalf("round-trip space contract=%+v", roundTrip)
	}
}
