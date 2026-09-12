package transfer

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	transferconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer/transfer_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// deviceTokenEnvVar is the fallback source for the hub device token when
// --device-token is not passed. A device exports it once after pairing
// (`devices redeem`/`devices request` print the value).
const deviceTokenEnvVar = "DEVICE_SYNC_HUB_DEVICE_TOKEN" //nolint:gosec // env var NAME that carries the token, not a hardcoded token value

// deviceTokenHeader is the wire header every transfer call presents. It is the
// device-trust credential, distinct from the owner JWT (Authorization).
const deviceTokenHeader = "X-Device-Token" //nolint:gosec // HTTP header name, not a hardcoded credential

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client transferconnect.TransferServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: transferconnect.NewTransferServiceClient(httpClient, baseURL),
	}
}

// deviceToken resolves the hub device token from --device-token, falling back to
// $DEVICE_SYNC_HUB_DEVICE_TOKEN. It is an error to call transfer without one:
// the server denies a tokenless request, so failing here yields a clearer
// message than a 401.
func deviceToken(ctx cliapp.RunContext) (string, error) {
	if tok := strings.TrimSpace(ctx.Flag("device-token")); tok != "" {
		return tok, nil
	}
	if tok := strings.TrimSpace(os.Getenv(deviceTokenEnvVar)); tok != "" {
		return tok, nil
	}
	return "", fmt.Errorf("no device token: pass --device-token or set $%s (obtain one via `device-sync-hub devices redeem`)", deviceTokenEnvVar)
}

// authed stamps the device token onto a Connect request header. Every transfer
// RPC is device-token authed, not owner-JWT authed.
func authed[T any](req *connect.Request[T], token string) *connect.Request[T] {
	req.Header().Set(deviceTokenHeader, token)
	return req
}

// sendText calls TransferService.CreateTextItem.
func (h *handlers) sendText(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	retention, err := parseRetention(ctx.Flag("retention"))
	if err != nil {
		return err
	}
	req := connect.NewRequest(&transferv1.CreateTextItemRequest{
		Text:           ctx.Positional("text"),
		Name:           strings.TrimSpace(ctx.Flag("name")),
		Retention:      retention,
		TargetDeviceId: strings.TrimSpace(ctx.Flag("target")),
	})
	resp, err := h.client.CreateTextItem(context.Background(), authed(req, token))
	if err != nil {
		return cliapp.WrapAPIError("send text", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Item == nil {
		return fmt.Errorf("server returned no item")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Sent text item %s.", resp.Msg.Item.Id)},
		Changes: []string{formatItem(resp.Msg.Item)},
		NextCommand: []string{
			"`transfer list` — see it from another device",
			fmt.Sprintf("`transfer get %s` — show this item", resp.Msg.Item.Id),
		},
	})
}

// list calls TransferService.ListItems with optional query/kind filters.
func (h *handlers) list(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	kind, err := parseKind(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	req := connect.NewRequest(&transferv1.ListItemsRequest{
		Query: strings.TrimSpace(ctx.Flag("query")),
		Kind:  kind,
	})
	resp, err := h.client.ListItems(context.Background(), authed(req, token))
	if err != nil {
		return cliapp.WrapAPIError("list items", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no items response")
	}
	results := make([]string, 0, len(resp.Msg.Items))
	for _, it := range resp.Msg.Items {
		results = append(results, formatItem(it))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d item(s).", len(resp.Msg.Items))},
		ResultsHeading: "Items",
		Results:        results,
		RetrievalHints: []string{
			"`transfer download <id> --out <path>` — pull an item's bytes",
			"`transfer get <id>` — show a single item's metadata",
			"`transfer send-text <text>` — send a snippet back",
		},
	})
}

// get calls TransferService.GetItem for a single item id.
func (h *handlers) get(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	id := ctx.Positional("id")
	req := connect.NewRequest(&transferv1.GetItemRequest{Id: id})
	resp, err := h.client.GetItem(context.Background(), authed(req, token))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get item %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Item == nil {
		return fmt.Errorf("server returned no item")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched item %s.", resp.Msg.Item.Id)},
		ResultsHeading: "Item",
		Results:        []string{formatItem(resp.Msg.Item)},
	})
}

// delete calls TransferService.DeleteItem.
func (h *handlers) delete(ctx cliapp.RunContext) error {
	token, err := deviceToken(ctx)
	if err != nil {
		return err
	}
	id := ctx.Positional("id")
	req := connect.NewRequest(&transferv1.DeleteItemRequest{Id: id})
	resp, err := h.client.DeleteItem(context.Background(), authed(req, token))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete item %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no delete response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted item %s.", resp.Msg.Id)},
		NextCommand: []string{"`transfer list` — confirm it's gone"},
	})
}

// parseRetention maps a CLI token to the proto Retention enum. An empty value
// yields RETENTION_UNSPECIFIED so the server applies the global default; an
// unknown value is a usage error rather than a silent default.
func parseRetention(v string) (transferv1.Retention, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return transferv1.Retention_RETENTION_UNSPECIFIED, nil
	case "live":
		return transferv1.Retention_RETENTION_LIVE, nil
	case "held":
		return transferv1.Retention_RETENTION_HELD, nil
	case "pinned":
		return transferv1.Retention_RETENTION_PINNED, nil
	default:
		return 0, fmt.Errorf("unknown retention %q (use one of: live, held, pinned)", v)
	}
}

// parseKind maps a CLI token to the proto ItemKind enum. Empty yields
// ITEM_KIND_UNSPECIFIED (both kinds); an unknown value is a usage error.
func parseKind(v string) (transferv1.ItemKind, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return transferv1.ItemKind_ITEM_KIND_UNSPECIFIED, nil
	case "text":
		return transferv1.ItemKind_ITEM_KIND_TEXT, nil
	case "file":
		return transferv1.ItemKind_ITEM_KIND_FILE, nil
	default:
		return 0, fmt.Errorf("unknown kind %q (use one of: text, file)", v)
	}
}

// retentionLabel renders the lifetime policy as a short human token.
func retentionLabel(r transferv1.Retention) string {
	switch r {
	case transferv1.Retention_RETENTION_LIVE:
		return "live"
	case transferv1.Retention_RETENTION_HELD:
		return "held"
	case transferv1.Retention_RETENTION_PINNED:
		return "pinned"
	default:
		return "default"
	}
}

// kindLabel renders the item kind as a short human token.
func kindLabel(k transferv1.ItemKind) string {
	switch k {
	case transferv1.ItemKind_ITEM_KIND_TEXT:
		return "text"
	case transferv1.ItemKind_ITEM_KIND_FILE:
		return "file"
	default:
		return "unknown"
	}
}

// formatItem produces a one-line representation suitable for both ListReport and
// MutationReport result blocks.
func formatItem(it *transferv1.Item) string {
	if it == nil {
		return "(nil)"
	}
	name := it.Name
	if name == "" {
		name = "(unnamed)"
	}
	target := "broadcast"
	if it.TargetDeviceId != "" {
		target = "→" + it.TargetDeviceId
	}
	created := ""
	if it.CreatedAt != nil {
		created = it.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [%s, %s, %dB, %s, %s]",
		it.Id, name, kindLabel(it.Kind), retentionLabel(it.Retention), it.SizeBytes, target, created)
}

// itemContentURL builds the absolute REST URL for an item's byte content.
func (h *handlers) itemContentURL(id string, thumb bool) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(h.core.APIBase()), "/")
	if base == "" {
		return "", fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	u := base + "/transfer/items/" + url.PathEscape(id) + "/content"
	if thumb {
		u += "?thumb=1"
	}
	return u, nil
}

// httpClient returns a plain HTTP client honoring the scenario's configured
// timeout, for the two REST byte edges (upload/download) that ride outside
// Connect.
func (h *handlers) httpClient() *http.Client {
	c := &http.Client{}
	if h.core != nil && h.core.HTTPClient != nil {
		c.Timeout = h.core.HTTPClient.Timeout()
	}
	return c
}

// filenameFromDisposition extracts the server-provided original filename from a
// Content-Disposition header, returning "" when absent or unparseable.
func filenameFromDisposition(header string) string {
	if strings.TrimSpace(header) == "" {
		return ""
	}
	if _, params, err := mime.ParseMediaType(header); err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return filepath.Base(name)
		}
	}
	return ""
}

// resolveOutputPath turns the --out flag plus the server's suggested filename
// into a concrete destination path. --out may be empty (cwd), a directory
// (join the suggested name), or a full file path (used verbatim).
func resolveOutputPath(out, suggested, id string) string {
	suggested = filepath.Base(strings.TrimSpace(suggested))
	if suggested == "" || suggested == "." || suggested == string(filepath.Separator) {
		suggested = id
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return suggested
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, suggested)
	}
	if strings.HasSuffix(out, string(filepath.Separator)) {
		return filepath.Join(out, suggested)
	}
	return out
}
