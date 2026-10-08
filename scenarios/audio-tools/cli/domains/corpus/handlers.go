package corpus

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus/corpus_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client corpusconnect.CorpusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: corpusconnect.NewCorpusServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	limit := 0
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--limit must be an integer: %q", v)
		}
		limit = n
	}
	resp, err := h.client.ListClips(context.Background(), connect.NewRequest(&corpusv1.ListClipsRequest{
		TagContains: ctx.Flag("tag"),
		Limit:       int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list-clips", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	clips := resp.Msg.GetClips()
	if len(clips) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No clips in the corpus. Record some in the Dictation Studio UI or `corpus import` a PCM file.")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%-38s  %-9s  %-7s  %-8s  %s\n", "ID", "SOURCE", "DUR_MS", "TAGS", "REFERENCE")
	for _, c := range clips {
		ref := c.GetReferenceText()
		if len(ref) > 48 {
			ref = ref[:45] + "..."
		}
		fmt.Fprintf(ctx.Stdout(), "%-38s  %-9s  %-7d  %-8s  %s\n",
			c.GetId(), sourceLabel(c.GetSource()), c.GetDurationMs(), strings.Join(c.GetTags(), ","), ref)
	}
	return nil
}

func (h *handlers) importClip(ctx cliapp.RunContext) error {
	audioPath := ctx.Positional("audio-file")
	if audioPath == "" {
		return fmt.Errorf("audio-file is required")
	}
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		return fmt.Errorf("read audio file: %w", err)
	}
	reference := ctx.Flag("reference")
	if rf := strings.TrimSpace(ctx.Flag("reference-file")); rf != "" {
		b, err := os.ReadFile(rf)
		if err != nil {
			return fmt.Errorf("read reference file: %w", err)
		}
		reference = string(b)
	}
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return fmt.Errorf("a reference transcript is required (--reference or --reference-file)")
	}

	format := strings.TrimSpace(ctx.Flag("format"))
	if format == "" {
		format = "pcm_s16le"
	}
	sampleRate := 16000
	if v := strings.TrimSpace(ctx.Flag("sample-rate")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--sample-rate must be an integer: %q", v)
		}
		sampleRate = n
	}
	source := corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM
	if ctx.Flag("scripted") == "true" {
		source = corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED
	}
	var tags []string
	if t := strings.TrimSpace(ctx.Flag("tags")); t != "" {
		for _, tag := range strings.Split(t, ",") {
			if s := strings.TrimSpace(tag); s != "" {
				tags = append(tags, s)
			}
		}
	}
	// Derive duration from PCM byte length for s16le mono.
	var durationMs int64
	if format == "pcm_s16le" || format == "pcm" {
		durationMs = int64(len(audio)) * 1000 / int64(sampleRate*2)
	}

	resp, err := h.client.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{
		Audio:         audio,
		ReferenceText: reference,
		Tags:          tags,
		DurationMs:    durationMs,
		SampleRateHz:  int32(sampleRate),
		Format:        format,
		Source:        source,
	}))
	if err != nil {
		return cliapp.WrapAPIError("create-clip", err, nil)
	}
	fmt.Fprintf(ctx.Stdout(), "Imported clip %s (%d bytes, %dms, %s).\n",
		resp.Msg.GetClip().GetId(), len(audio), resp.Msg.GetClip().GetDurationMs(), format)
	return nil
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetClip(context.Background(), connect.NewRequest(&corpusv1.GetClipRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("get-clip", err, nil)
	}
	c := resp.Msg.GetClip()
	fmt.Fprintf(ctx.Stdout(), "id             = %s\n", c.GetId())
	fmt.Fprintf(ctx.Stdout(), "source         = %s\n", sourceLabel(c.GetSource()))
	fmt.Fprintf(ctx.Stdout(), "duration_ms    = %d\n", c.GetDurationMs())
	fmt.Fprintf(ctx.Stdout(), "sample_rate_hz = %d\n", c.GetSampleRateHz())
	fmt.Fprintf(ctx.Stdout(), "format         = %s\n", c.GetFormat())
	fmt.Fprintf(ctx.Stdout(), "tags           = %s\n", strings.Join(c.GetTags(), ", "))
	fmt.Fprintf(ctx.Stdout(), "blob_key       = %s\n", c.GetBlobKey())
	fmt.Fprintf(ctx.Stdout(), "reference      = %s\n", c.GetReferenceText())
	return nil
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	_, err := h.client.DeleteClip(context.Background(), connect.NewRequest(&corpusv1.DeleteClipRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("delete-clip", err, nil)
	}
	fmt.Fprintf(ctx.Stdout(), "Deleted clip %s.\n", id)
	return nil
}

func sourceLabel(s corpusv1.ClipSource) string {
	switch s {
	case corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED:
		return "scripted"
	default:
		return "free_form"
	}
}
