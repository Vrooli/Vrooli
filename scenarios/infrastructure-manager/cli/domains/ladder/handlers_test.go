package ladder

import (
	"strings"
	"testing"

	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
)

func TestFormatCellDistinguishesUnsampledHost(t *testing.T) {
	got := formatCell(&ladderv1.LadderCell{
		CellRef:     "substrate/SB11",
		Key:         "thermal-sensor/telemetry/macos",
		Reason:      "no live join produced a grade for this cell",
		ReasonCode:  "host_not_sampled",
		Observation: ladderv1.Observation_OBSERVATION_UNREAD,
		Trust:       ladderv1.TrustVerdict_TRUST_VERDICT_UNTRUSTED,
	})
	if !strings.Contains(got, "host_not_sampled:") {
		t.Fatalf("ladder CLI omitted unsampled-host reason code: %q", got)
	}
	if strings.Contains(got, "source_down") {
		t.Fatalf("ladder CLI mislabeled an unsampled host as source_down: %q", got)
	}
}
