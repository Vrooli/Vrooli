package walletrecovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	attestPath           = "/api/wallet-recovery/attest"
	finishPath           = "/api/wallet-recovery/reservation/finish"
	readLeaseAcquirePath = "/api/wallet-recovery/read-lease/acquire"
	readLeaseRenewPath   = "/api/wallet-recovery/read-lease/renew"
	readLeaseReleasePath = "/api/wallet-recovery/read-lease/release"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client interface {
		Do(*http.Request) (*http.Response, error)
	}
	baseURL string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: client, baseURL: strings.TrimRight(base, "/")}
}

func (h *handlers) attest(ctx cliapp.RunContext) error {
	return h.post(ctx, attestPath, map[string]string{"reservation_id": ctx.Flag("reservation-id"), "target_id": ctx.Flag("target-id"), "destination_id": ctx.Flag("destination-id")}, "request wallet recovery attestation")
}

func (h *handlers) finish(ctx cliapp.RunContext) error {
	return h.post(ctx, finishPath, map[string]string{"reservation_id": ctx.Flag("reservation-id"), "target_id": ctx.Flag("target-id"), "state": ctx.Flag("state")}, "finalize wallet recovery reservation")
}

func (h *handlers) readLeaseAcquire(ctx cliapp.RunContext) error {
	return h.post(ctx, readLeaseAcquirePath, map[string]string{"lease_id": ctx.Flag("lease-id"), "target_name": ctx.Flag("target-name"), "destination_id": ctx.Flag("destination-id"), "snapshot_id": ctx.Flag("snapshot-id")}, "acquire wallet recovery read lease")
}

func (h *handlers) readLeaseRenew(ctx cliapp.RunContext) error {
	return h.post(ctx, readLeaseRenewPath, map[string]string{"lease_id": ctx.Flag("lease-id"), "target_id": ctx.Flag("target-id")}, "renew wallet recovery read lease")
}

func (h *handlers) readLeaseRelease(ctx cliapp.RunContext) error {
	return h.post(ctx, readLeaseReleasePath, map[string]string{"lease_id": ctx.Flag("lease-id"), "target_id": ctx.Flag("target-id")}, "release wallet recovery read lease")
}

func (h *handlers) post(ctx cliapp.RunContext, path string, payload any, op string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, h.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return cliapp.WrapAPIError(op, err, nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: DBM returned HTTP %d", op, resp.StatusCode)
	}
	var response json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("%s: decode DBM response: %w", op, err)
	}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), response)
	}
	var values map[string]any
	if err := json.Unmarshal(response, &values); err != nil {
		return err
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{op + " completed."}, Changes: []string{fmt.Sprintf("%v", values)}})
}
