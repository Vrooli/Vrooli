package setup

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func TestOperatorChoicesRenderDistinctly(t *testing.T) {
	cases := []struct {
		name    string
		choice  hostreqspec.OperatorChoice
		blocker hostreqkit.BlockingReason
		want    string
	}{
		{name: "opted in", choice: hostreqspec.OperatorChoiceOptedIn, want: "operator choice: opted_in"},
		{name: "declined", choice: hostreqspec.OperatorChoiceDeclined, blocker: hostreqkit.BlockingOperatorDeclined, want: "operator choice: declined"},
		{name: "not recorded", choice: hostreqspec.OperatorChoiceNotRecorded, blocker: hostreqkit.BlockingOperatorChoiceMissing, want: "operator choice: not_recorded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := vrooliruntime.ItemStatus{
				Name:           "example_safeguard",
				Kind:           hostreq.KindSafeguard,
				Required:       false,
				OperatorChoice: tc.choice,
				BlockingReason: tc.blocker,
				ExecutionState: vrooliruntime.ExecutionPending,
			}
			var output strings.Builder
			renderRequirementVerboseItem(&output, item, false)
			if !strings.Contains(output.String(), tc.want) {
				t.Fatalf("output missing %q:\n%s", tc.want, output.String())
			}
		})
	}
}
