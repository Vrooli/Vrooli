package archive

import (
	"fmt"
	"os"
	"strings"

	"file-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `archive` subcommand group for archive-style operations:
// compress, extract, split, merge. These wrap the POST endpoints under
// `/api/v1/files/`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "archive",
		Description: "Archive and chunking operations (compress, extract, split, merge)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "compress", Description: "Compress files into an archive", Run: func(args []string) error { return runCompress(core, args) }},
			{Name: "extract", Description: "Extract files from an archive", Run: func(args []string) error { return runExtract(core, args) }},
			{Name: "split", Description: "Split a file into parts", Run: func(args []string) error { return runSplit(core, args) }},
			{Name: "merge", Description: "Merge file parts into a single output", Run: func(args []string) error { return runMerge(core, args) }},
		},
	}
}

func runCompress(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("archive compress")
	output := fs.String("output", "", "Output archive path (required)")
	format := fs.String("format", "zip", "Archive format: zip|tar|gzip")
	level := fs.Int("level", 6, "Compression level 0-9")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	files := fs.Args()
	if len(files) == 0 {
		return fmt.Errorf("usage: archive compress <file>... --output PATH [--format zip|tar|gzip] [--level N]")
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output PATH is required")
	}

	payload := map[string]interface{}{
		"files":             files,
		"archive_format":    *format,
		"output_path":       *output,
		"compression_level": *level,
	}

	body, err := core.Request("POST", "/files/compress", nil, payload)
	if err != nil {
		return err
	}
	var resp support.CompressResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Archive: %s", resp.ArchivePath),
		fmt.Sprintf("Files included: %d", resp.FilesIncluded),
		fmt.Sprintf("Original size: %d bytes", resp.OriginalSizeBytes),
		fmt.Sprintf("Compressed size: %d bytes", resp.CompressedSize),
		fmt.Sprintf("Ratio: %.2f", resp.CompressionRatio),
	}
	if resp.Checksum != "" {
		changes = append(changes, fmt.Sprintf("Checksum: %s", resp.Checksum))
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Compression complete (op %s)", resp.OperationID)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s archive extract --archive %s --dest <dir>", support.CLIName, resp.ArchivePath),
			fmt.Sprintf("%s files checksum %s", support.CLIName, resp.ArchivePath),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExtract(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("archive extract")
	archive := fs.String("archive", "", "Archive path (required)")
	dest := fs.String("dest", ".", "Destination directory")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*archive) == "" {
		return fmt.Errorf("--archive PATH is required")
	}

	payload := map[string]interface{}{
		"archive_path":     *archive,
		"destination_path": *dest,
	}
	body, err := core.Request("POST", "/files/extract", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ExtractResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Destination: %s", *dest),
		fmt.Sprintf("Extracted files: %d", resp.TotalFiles),
		fmt.Sprintf("Total size: %d bytes", resp.TotalSizeBytes),
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Extraction complete (op %s)", resp.OperationID)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s files metadata --path %s", support.CLIName, *dest)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSplit(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("archive split")
	file := fs.String("file", "", "File to split (required)")
	size := fs.Int64("size", 0, "Chunk size in bytes")
	parts := fs.Int("parts", 0, "Number of parts")
	pattern := fs.String("output-pattern", "", "Output file pattern (defaults to <file>.part)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*file) == "" {
		return fmt.Errorf("--file PATH is required")
	}
	if *size == 0 && *parts == 0 {
		return fmt.Errorf("specify either --size BYTES or --parts N")
	}

	payload := map[string]interface{}{
		"file":           *file,
		"size":           *size,
		"parts":          *parts,
		"output_pattern": *pattern,
	}
	body, err := core.Request("POST", "/files/split", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SplitResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Parts created: %d", resp.TotalParts),
		fmt.Sprintf("Chunk size: %d bytes", resp.ChunkSize),
	}
	for _, p := range resp.Parts {
		changes = append(changes, "part: "+p)
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Split complete (op %s)", resp.OperationID)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s archive merge --pattern '%s.*' --output <file>", support.CLIName, *file)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runMerge(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("archive merge")
	pattern := fs.String("pattern", "", "Glob pattern for part files (required)")
	output := fs.String("output", "", "Output file path (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*pattern) == "" || strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--pattern and --output are required")
	}

	payload := map[string]interface{}{
		"pattern": *pattern,
		"output":  *output,
	}
	body, err := core.Request("POST", "/files/merge", nil, payload)
	if err != nil {
		return err
	}
	var resp support.MergeResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Output: %s", resp.OutputFile),
		fmt.Sprintf("Merged parts: %d", resp.MergedParts),
		fmt.Sprintf("Total size: %d bytes", resp.TotalSize),
	}
	if resp.Checksum != "" {
		changes = append(changes, fmt.Sprintf("Checksum (sha256): %s", resp.Checksum))
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Merge complete (op %s)", resp.OperationID)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s files metadata --path %s", support.CLIName, resp.OutputFile)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
