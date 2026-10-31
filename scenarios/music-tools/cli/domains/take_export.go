package domains

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Binary exports reuse the existing attributed API client. A staged file plus
// no-replace hard-link publication preserves old outputs and never exposes a
// partially downloaded/written new output as final.
func exportTake(app *cliapp.ScenarioApp, args []string, audio bool) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("take ID is required before flags")
	}
	id := args[0]
	fs := flag.NewFlagSet("takes export", flag.ContinueOnError)
	out := fs.String("out", "", "new output file (required for audio)")
	if e := fs.Parse(args[1:]); e != nil {
		return e
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected export arguments")
	}
	if audio && *out == "" {
		return fmt.Errorf("audio requires --out NEW.wav")
	}
	var stage *os.File
	if *out != "" {
		if _, e := os.Lstat(*out); !os.IsNotExist(e) {
			return fmt.Errorf("output must be a new file")
		}
		var e error
		stage, e = os.CreateTemp(filepath.Dir(*out), ".music-export-")
		if e != nil {
			return e
		}
		defer func() { stage.Close(); os.Remove(stage.Name()) }()
	}
	suffix := "/receipt"
	if audio {
		suffix = "/audio"
	}
	raw, e := app.APIClient.WithoutRedirects().Get(app.APIPath("/takes/"+url.PathEscape(id)+suffix), nil)
	if e != nil {
		return e
	}
	if audio {
		if len(raw) < 44 || len(raw) > 128<<20 || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WAVE" || uint64(binary.LittleEndian.Uint32(raw[4:8]))+8 != uint64(len(raw)) {
			return fmt.Errorf("take audio is not a complete supported WAV")
		}
	} else if len(raw) > 512<<10 {
		return fmt.Errorf("receipt exceeds512KiB")
	}
	if stage == nil {
		_, e = os.Stdout.Write(raw)
		return e
	}
	if _, e = stage.Write(raw); e != nil {
		return e
	}
	if e = stage.Sync(); e != nil {
		return e
	}
	if e = stage.Close(); e != nil {
		return e
	}
	if e = os.Link(stage.Name(), *out); e != nil {
		return fmt.Errorf("publish new output without replacement: %w", e)
	}
	return printJSON([]byte(fmt.Sprintf(`{"take_id":%q,"output":%q}`, id, *out)))
}
