package support

import (
	"flag"
	"strings"
	"testing"
)

func TestFlagHelpIsGeneratedFromTheHandlersOwnFlagSet(t *testing.T) {
	ran := false
	handler := func(args []string) error {
		fs := flag.NewFlagSet("demo", flag.ContinueOnError)
		fs.Bool("quiet", false, "Output only IDs")
		fs.Int("limit", 50, "Maximum rows")
		fs.String("json", "", "reserved")
		if err := ParseFlags(fs, args); err != nil {
			return err
		}
		ran = true
		return nil
	}
	help := FlagHelp(handler)
	for _, want := range []string{"--quiet ", "Output only IDs", "--limit <value>", "Maximum rows (default: 50)"} {
		if !strings.Contains(help, want) {
			t.Errorf("help missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "--quiet <value>") {
		t.Errorf("bool flag must not take a value:\n%s", help)
	}
	if ran {
		t.Fatal("describing flags must not run the handler")
	}
	if err := handler([]string{"--limit", "3"}); err != nil || !ran {
		t.Fatalf("normal parsing must still run the handler: %v", err)
	}
}
