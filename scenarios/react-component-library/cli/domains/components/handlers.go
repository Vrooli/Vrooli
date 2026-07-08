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
		Match:    ctx.Flag("match"),
		Tag:      ctx.Flag("tag"),
		Category: ctx.Flag("category"),
		StyleId:  ctx.Flag("style"),
		Affinity: ctx.Flag("affinity"),
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		// Comma-separated multi-tag OR. Trim entries silently — the
		// repository drops blanks too, so `--tags ,form,` is fine.
		for _, t := range strings.Split(rawTags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				req.Tags = append(req.Tags, trimmed)
			}
		}
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

func (h *handlers) styles(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDesignStyles(context.Background(), connect.NewRequest(&componentsv1.ListDesignStylesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list design styles", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no styles response")
	}
	results := make([]string, 0, len(resp.Msg.Styles))
	for _, style := range resp.Msg.Styles {
		results = append(results, fmt.Sprintf("%s\t%s\t%s", style.Id, style.Name, strings.Join(style.Supports, ",")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d design style(s).", len(resp.Msg.Styles))},
		ResultsHeading: "Design styles",
		Results:        results,
	})
}

func (h *handlers) init(ctx cliapp.RunContext) error {
	req := &componentsv1.InitializeComponentRequest{
		Slug:           ctx.Positional("slug"),
		LibraryId:      ctx.Flag("library-id"),
		DisplayName:    ctx.Flag("display-name"),
		Description:    ctx.Flag("description"),
		InitialVersion: ctx.Flag("version"),
		FileName:       ctx.Flag("file-name"),
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		req.Tags = splitCSV(rawTags)
	}
	if src := ctx.Flag("source-file"); src != "" {
		body, err := readSourceArg(src)
		if err != nil {
			return err
		}
		req.InitialSource = string(body)
	}
	resp, err := h.client.InitializeComponent(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("initialize component", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no initialize response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Initialized %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Created files",
		Results:        []string{resp.Msg.ManifestPath, resp.Msg.SourcePath},
		RetrievalHints: []string{
			"`components content-get " + resp.Msg.Component.Id + "` — inspect the generated source",
		},
	})
}

func (h *handlers) versionCreate(ctx cliapp.RunContext) error {
	req := &componentsv1.CreateComponentVersionRequest{
		ComponentId: ctx.Positional("component-id"),
		Version:     ctx.Positional("version"),
		FromVersion: ctx.Flag("from-version"),
		FileName:    ctx.Flag("file-name"),
		ChangelogMd: ctx.Flag("changelog"),
	}
	switch {
	case ctx.Flag("draft") == "true":
		req.Intent = componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_DRAFT
	case ctx.Flag("release") == "true":
		req.Intent = componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_RELEASE
	}
	if src := ctx.Flag("source-file"); src != "" {
		body, err := readSourceArg(src)
		if err != nil {
			return err
		}
		req.Source = string(body)
	}
	resp, err := h.client.CreateComponentVersion(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("create component version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil {
		return fmt.Errorf("server returned no version create response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Created version %s.", resp.Msg.Version.Version)},
		ResultsHeading: "Source",
		Results:        []string{resp.Msg.SourcePath},
	})
}

func (h *handlers) manifestUpdate(ctx cliapp.RunContext) error {
	req := &componentsv1.UpdateComponentManifestRequest{
		ComponentId:   ctx.Positional("component-id"),
		DisplayName:   ctx.Flag("display-name"),
		Description:   ctx.Flag("description"),
		LatestVersion: ctx.Flag("latest-version"),
		DraftVersion:  ctx.Flag("draft-version"),
	}
	if rawTags := ctx.Flag("tags"); rawTags != "" {
		req.Tags = splitCSV(rawTags)
	}
	if raw := ctx.Flag("deprecated-versions"); raw != "" {
		req.DeprecatedVersions = splitCSV(raw)
	}
	resp, err := h.client.UpdateComponentManifest(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("update component manifest", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Component == nil {
		return fmt.Errorf("server returned no manifest update response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Updated %s.", resp.Msg.Component.LibraryId)},
		ResultsHeading: "Component",
		Results:        []string{formatComponent(resp.Msg.Component)},
	})
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func readSourceArg(src string) ([]byte, error) {
	if src == "-" {
		return readAllStdin()
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read source %q: %w", src, err)
	}
	return body, nil
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
		Summary:        []string{fmt.Sprintf("Read %s (sha256=%s).", resp.Msg.SourcePath, resp.Msg.Sha256)},
		ResultsHeading: "Content",
		Results:        []string{resp.Msg.Content},
	})
}

func (h *handlers) versions(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	req := &componentsv1.ListComponentVersionsRequest{ComponentId: componentID}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListComponentVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list versions for component %q", componentID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no versions response")
	}
	results := make([]string, 0, len(resp.Msg.Versions))
	for _, v := range resp.Msg.Versions {
		results = append(results, formatVersion(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d version(s).", len(resp.Msg.Versions))},
		ResultsHeading: "Versions",
		Results:        results,
	})
}

func (h *handlers) showVersion(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	version := ctx.Positional("version")
	resp, err := h.client.GetComponentVersionContent(context.Background(), connect.NewRequest(&componentsv1.GetComponentVersionContentRequest{
		ComponentId: componentID,
		Version:     version,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show version %q for component %q", version, componentID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no version response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Read %s.", resp.Msg.Version.SourcePath)},
		ResultsHeading: "Content",
		Results:        []string{resp.Msg.Content},
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
	slotPart := ""
	if c.Slot != "" {
		slotPart = " slot=" + c.Slot
	}
	stylePart := ""
	if len(c.DesignStyles) > 0 {
		parts := make([]string, 0, len(c.DesignStyles))
		for _, style := range c.DesignStyles {
			parts = append(parts, fmt.Sprintf("%s:%s", style.StyleId, formatDesignAffinity(style.Affinity)))
		}
		stylePart = " styles=[" + strings.Join(parts, ",") + "]"
	}
	return fmt.Sprintf("%s — %s%s%s%s%s @ %s [indexed=%s]", c.LibraryId, c.DisplayName, versionPart, slotPart, stylePart, tagsPart, c.SourcePath, indexed)
}

func formatDesignAffinity(affinity componentsv1.DesignAffinity) string {
	switch affinity {
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_NATIVE:
		return "native"
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_COMPATIBLE:
		return "compatible"
	case componentsv1.DesignAffinity_DESIGN_AFFINITY_DISCOURAGED:
		return "discouraged"
	default:
		return "unspecified"
	}
}

func formatVersion(v *componentsv1.ComponentVersion) string {
	if v == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s %s sha=%s @ %s", v.Id, v.Version, v.Status.String(), v.ContentSha256, v.SourcePath)
}
