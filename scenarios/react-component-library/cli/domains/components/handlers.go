package components

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client, mirroring the cli/domains/notes/ shape.
type handlers struct {
	core   *cliapp.ScenarioApp
	client componentsconnect.ComponentsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: componentsconnect.NewComponentsServiceClient(httpClient, baseURL),
	}
}

// index calls ComponentsService.IndexComponents. The walk runs server-side;
// the response carries the summary.
func (h *handlers) index(ctx cliapp.RunContext) error {
	resp, err := h.client.IndexComponents(context.Background(), connect.NewRequest(&componentsv1.IndexComponentsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("index components", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no index response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf(
		"Scanned %d file(s); indexed %d, skipped %d, deleted %d.",
		msg.Scanned, msg.Indexed, msg.Skipped, msg.Deleted)}
	results := append([]string{}, msg.LibraryIds...)
	if len(msg.Errors) > 0 {
		summary = append(summary, fmt.Sprintf("%d error(s) reported.", len(msg.Errors)))
		results = append(results, msg.Errors...)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Library IDs",
		Results:        results,
		RetrievalHints: []string{
			"`components list` — show indexed components",
			"`components get-by-library-id <libraryId>` — inspect a single entry",
		},
	})
}

// list calls ComponentsService.ListComponents with optional filter flags.
func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &componentsv1.ListComponentsRequest{
		Match: ctx.Flag("match"),
		Tag:   ctx.Flag("tag"),
	}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponents(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list components", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list response")
	}
	results := make([]string, 0, len(resp.Msg.Components))
	for _, c := range resp.Msg.Components {
		results = append(results, formatComponent(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d component(s).", len(resp.Msg.Components))},
		ResultsHeading: "Components",
		Results:        results,
		RetrievalHints: []string{
			"`components get <id>` — show a single component",
			"`components index` — refresh from disk",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetComponent(context.Background(), connect.NewRequest(&componentsv1.GetComponentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched component %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
}

func (h *handlers) getByLibraryID(ctx cliapp.RunContext) error {
	libid := ctx.Positional("library-id")
	resp, err := h.client.GetComponentByLibraryId(context.Background(), connect.NewRequest(&componentsv1.GetComponentByLibraryIdRequest{LibraryId: libid}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get component %q", libid), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no component")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched component %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
}

// contentGet calls ComponentsService.GetComponentContent and prints the
// source body to stdout. Human output writes the body as-is so it can
// be piped (e.g. `… content get <id> > Button.tsx`); --json emits the
// proto wire shape with body, source_path, and sha256.
func (h *handlers) contentGet(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetComponentContent(context.Background(), connect.NewRequest(&componentsv1.GetComponentContentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get content for component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no content response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Read %s (sha256=%s).", resp.Msg.SourcePath, resp.Msg.Sha256)},
		ResultsHeading: "Content",
		Results: []string{resp.Msg.Content},
	})
}

// contentSet reads <file> from disk (or "-" for stdin) and calls
// ComponentsService.UpdateComponentContent. --expected-sha256 forwards
// the optimistic-concurrency guard.
func (h *handlers) contentSet(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	src := ctx.Positional("file")
	var body []byte
	var err error
	if src == "-" {
		body, err = readAllStdin()
	} else {
		body, err = os.ReadFile(src)
	}
	if err != nil {
		return fmt.Errorf("read source %q: %w", src, err)
	}
	req := &componentsv1.UpdateComponentContentRequest{
		Id:             id,
		Content:        string(body),
		ExpectedSha256: ctx.Flag("expected-sha256"),
	}
	resp, err := h.client.UpdateComponentContent(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set content for component %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no update response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Wrote %s (sha256=%s).", resp.Msg.SourcePath, resp.Msg.Sha256)},
		RetrievalHints: []string{
			"`components index` — re-walk if the @libraryId header changed",
		},
	})
}

func readAllStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// formatComponent produces a one-line representation for the result block.
func formatComponent(c *componentsv1.Component) string {
	if c == nil {
		return "(nil)"
	}
	indexed := ""
	if c.IndexedAt != nil {
		indexed = c.IndexedAt.AsTime().Format(time.RFC3339)
	}
	tagsPart := ""
	if len(c.Tags) > 0 {
		tagsPart = " tags=[" + strings.Join(c.Tags, ",") + "]"
	}
	versionPart := ""
	if c.Version != "" {
		versionPart = " v" + c.Version
	}
	return fmt.Sprintf("%s — %s%s%s @ %s [indexed=%s]", c.LibraryId, c.DisplayName, versionPart, tagsPart, c.SourcePath, indexed)
}
