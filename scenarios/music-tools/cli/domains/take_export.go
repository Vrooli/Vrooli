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

func exportAnalysisStem(app *cliapp.ScenarioApp, args []string) error {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("analysis job ID and stem name are required before flags")
	}
	id, stemName := args[0], args[1]
	switch stemName {
	case "vocals", "drums", "bass", "other":
	default:
		return fmt.Errorf("stem must be vocals, drums, bass or other")
	}
	fs := flag.NewFlagSet("analysis stem", flag.ContinueOnError)
	out := fs.String("out", "", "new output WAV file (required)")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() != 0 || *out == "" {
		return fmt.Errorf("--out NEW.wav is required and unexpected arguments are not accepted")
	}
	if _, err := os.Lstat(*out); !os.IsNotExist(err) {
		return fmt.Errorf("output must be a new file")
	}
	stage, err := os.CreateTemp(filepath.Dir(*out), ".music-analysis-stem-")
	if err != nil {
		return err
	}
	defer func() { stage.Close(); os.Remove(stage.Name()) }()
	raw, err := app.APIClient.WithoutRedirects().Get(app.APIPath("/analysis/"+url.PathEscape(id)+"/stems/"+url.PathEscape(stemName)), nil)
	if err != nil {
		return err
	}
	if len(raw) < 44 || len(raw) > 128<<20 || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WAVE" || uint64(binary.LittleEndian.Uint32(raw[4:8]))+8 != uint64(len(raw)) {
		return fmt.Errorf("stem is not a complete supported WAV within the 128 MiB limit")
	}
	if _, err = stage.Write(raw); err != nil {
		return err
	}
	if err = stage.Sync(); err != nil {
		return err
	}
	if err = stage.Close(); err != nil {
		return err
	}
	if err = os.Link(stage.Name(), *out); err != nil {
		return fmt.Errorf("publish new output without replacement: %w", err)
	}
	return printJSON([]byte(fmt.Sprintf(`{"job_id":%q,"stem":%q,"output":%q}`, id, stemName, *out)))
}

func exportSourceAudio(app *cliapp.ScenarioApp, args []string) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("source SHA-256 is required before flags")
	}
	digest := args[0]
	fs := flag.NewFlagSet("sources audio", flag.ContinueOnError)
	out := fs.String("out", "", "new audio file path (required)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 || *out == "" {
		return fmt.Errorf("--out NEW.audio is required and unexpected arguments are not accepted")
	}
	if _, err := os.Lstat(*out); !os.IsNotExist(err) {
		return fmt.Errorf("output must be a new file")
	}
	stage, err := os.CreateTemp(filepath.Dir(*out), ".music-source-")
	if err != nil {
		return err
	}
	defer func() { stage.Close(); os.Remove(stage.Name()) }()
	raw, err := app.APIClient.WithoutRedirects().Get(app.APIPath("/sources/"+url.PathEscape(digest)+"/audio"), nil)
	if err != nil {
		return err
	}
	if len(raw) == 0 || len(raw) > 128<<20 {
		return fmt.Errorf("source audio is empty or exceeds the 128 MiB limit")
	}
	if _, err = stage.Write(raw); err != nil {
		return err
	}
	if err = stage.Sync(); err != nil {
		return err
	}
	if err = stage.Close(); err != nil {
		return err
	}
	if err = os.Link(stage.Name(), *out); err != nil {
		return fmt.Errorf("publish new output without replacement: %w", err)
	}
	return printJSON([]byte(fmt.Sprintf(`{"sha256":%q,"output":%q}`, digest, *out)))
}
