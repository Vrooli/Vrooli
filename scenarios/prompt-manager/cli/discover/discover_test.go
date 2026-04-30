package discover

import "testing"

func TestFormatNumberAddsThousandsSeparators(t *testing.T) {
	cases := map[int]string{
		999:     "999",
		1200:    "1,200",
		1234567: "1,234,567",
	}
	for input, expected := range cases {
		if got := formatNumber(input); got != expected {
			t.Fatalf("formatNumber(%d) = %q, want %q", input, got, expected)
		}
	}
}

func TestCommandsRegistersDiscoverCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Discovery" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "discover" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}
