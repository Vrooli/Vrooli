package providers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the RegistryService client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client registryconnect.RegistryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: registryconnect.NewRegistryServiceClient(httpClient, baseURL),
	}
}

// register reads a ProviderDescriptor from --descriptor (raw JSON, or @path to
// read from a file) and upserts it via RegistryService.RegisterProvider.
func (h *handlers) register(ctx cliapp.RunContext) error {
	blob, err := readDescriptorArg(ctx.Flag("descriptor"))
	if err != nil {
		return err
	}
	desc := &registryv1.ProviderDescriptor{}
	if err := protojson.Unmarshal(blob, desc); err != nil {
		return fmt.Errorf("parse --descriptor JSON: %w", err)
	}

	resp, err := h.client.RegisterProvider(context.Background(), connect.NewRequest(&registryv1.RegisterProviderRequest{
		Descriptor_: desc,
	}))
	if err != nil {
		return cliapp.WrapAPIError("register provider", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetDescriptor_() == nil {
		return fmt.Errorf("server returned no descriptor")
	}

	verb := "Updated"
	if resp.Msg.GetCreated() {
		verb = "Registered"
	}
	d := resp.Msg.GetDescriptor_()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s provider %s.", verb, d.GetProviderId())},
		Changes: []string{formatProvider(d)},
		NextCommand: []string{
			"`providers list` — show all registered providers",
			fmt.Sprintf("`providers remove %s` — deregister this leaf", d.GetProviderId()),
		},
	})
}

// list returns registered providers, optionally filtered by --bucket, --type,
// and/or --state.
func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &registryv1.ListProvidersRequest{Type: strings.TrimSpace(ctx.Flag("type"))}

	bucket, err := parseBucket(ctx.Flag("bucket"))
	if err != nil {
		return err
	}
	req.Bucket = bucket

	state, err := parseState(ctx.Flag("state"))
	if err != nil {
		return err
	}
	req.State = state

	resp, err := h.client.ListProviders(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list providers", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no providers response")
	}

	results := make([]string, 0, len(resp.Msg.GetProviders()))
	for _, p := range resp.Msg.GetProviders() {
		results = append(results, formatProvider(p))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d provider(s).", len(resp.Msg.GetProviders()))},
		ResultsHeading: "Providers",
		Results:        results,
		RetrievalHints: []string{
			"`providers list --bucket DO|REUSE|KNOW|STATE` — filter by routing bucket",
			"`providers list --type <type>` — filter by leaf type (command, doc, record…)",
			"`providers list --state capability_gap` — show tracked gap stubs",
			"`providers register --descriptor @path.json` — register/update a provider",
		},
	})
}

// remove deregisters a provider leaf by provider_id.
func (h *handlers) remove(ctx cliapp.RunContext) error {
	id := ctx.Positional("provider_id")
	resp, err := h.client.DeregisterProvider(context.Background(), connect.NewRequest(&registryv1.DeregisterProviderRequest{
		ProviderId: id,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("deregister provider %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}

	result := fmt.Sprintf("No provider %q was registered (nothing to remove).", id)
	if resp.Msg.GetRemoved() {
		result = fmt.Sprintf("Deregistered provider %s.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: []string{"`providers list` — show all registered providers"},
	})
}

// readDescriptorArg returns the descriptor JSON bytes from a raw flag value.
// A leading '@' means "read from this file path"; otherwise the value is the
// JSON itself.
func readDescriptorArg(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--descriptor is required (JSON, or @path to a file)")
	}
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		blob, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read --descriptor file %q: %w", path, err)
		}
		return blob, nil
	}
	return []byte(raw), nil
}

// parseBucket maps a friendly bucket token (case-insensitive, with or without
// the BUCKET_ prefix) to the enum. Empty string means "no filter".
func parseBucket(s string) (registryv1.Bucket, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return registryv1.Bucket_BUCKET_UNSPECIFIED, nil
	}
	key := "BUCKET_" + strings.ToUpper(strings.TrimPrefix(strings.ToUpper(s), "BUCKET_"))
	if v, ok := registryv1.Bucket_value[key]; ok && v != 0 {
		return registryv1.Bucket(v), nil
	}
	return 0, fmt.Errorf("invalid --bucket %q (want one of DO, REUSE, KNOW, STATE)", s)
}

// parseState maps a friendly state token to the enum. Empty means "no filter".
func parseState(s string) (registryv1.ProviderState, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return registryv1.ProviderState_PROVIDER_STATE_UNSPECIFIED, nil
	}
	key := "PROVIDER_STATE_" + strings.ToUpper(strings.TrimPrefix(strings.ToUpper(s), "PROVIDER_STATE_"))
	if v, ok := registryv1.ProviderState_value[key]; ok && v != 0 {
		return registryv1.ProviderState(v), nil
	}
	return 0, fmt.Errorf("invalid --state %q (want active or capability_gap)", s)
}

// formatProvider produces a one-line representation for ListReport and
// MutationReport result blocks.
func formatProvider(p *registryv1.ProviderDescriptor) string {
	if p == nil {
		return "(nil)"
	}
	bucket := strings.TrimPrefix(p.GetBucket().String(), "BUCKET_")
	state := strings.TrimPrefix(p.GetState().String(), "PROVIDER_STATE_")
	tag := ""
	if p.GetState() == registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP {
		tag = fmt.Sprintf(" → gap (home: %s)", p.GetIntendedHome())
	}
	return fmt.Sprintf("%s [%s/%s] %s%s", p.GetProviderId(), bucket, p.GetType(), strings.ToLower(state), tag)
}
