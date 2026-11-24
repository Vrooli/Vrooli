package capacitysync

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

const defaultReservationTTL = time.Hour

var errReservationClaimNotReady = errors.New("Ollama's capacity companion has not admitted the model reservation yet")

// ReservationCommands exposes the Ollama-owned pre-load reservation handoff.
// The shared capacity companion remains the sole owner of the resource claim.
func ReservationCommands(h *Handlers) []cliapp.Command {
	if h == nil {
		h = Default()
	}
	return []cliapp.Command{
		{
			Name:        "reserve-model",
			Description: "Reserve a policy model's VRAM across startup and first load",
			Usage:       "resource-ollama capacity reserve-model --role <role> [--ttl-seconds 3600] [--json]",
			Run:         h.ReserveModel,
		},
		{
			Name:        "confirm-model-reservation",
			Description: "Bind an active model reservation to Ollama's admitted resource claim",
			Usage:       "resource-ollama capacity confirm-model-reservation --id <reservation-id> [--wait-seconds 30] [--json]",
			Run:         h.ConfirmModelReservation,
		},
		{
			Name:        "release-model-reservation",
			Description: "Hand a loaded model reservation back to Ollama's footprint companion",
			Usage:       "resource-ollama capacity release-model-reservation --id <reservation-id> [--json]",
			Run:         h.ReleaseModelReservation,
		},
	}
}

func (h *Handlers) ReserveModel(args []string) error {
	fs := flag.NewFlagSet("capacity reserve-model", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Ollama policy role to reserve")
	ttlSeconds := fs.Int("ttl-seconds", int(defaultReservationTTL.Seconds()), "Reservation safety expiry")
	asJSON := fs.Bool("json", false, "Emit a machine-readable reservation receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if strings.TrimSpace(*role) == "" || *ttlSeconds <= 0 {
		return errors.New("--role and a positive --ttl-seconds are required")
	}
	p, _, err := policy.LoadDefaultFile(h.getenv())
	if err != nil {
		return fmt.Errorf("load Ollama model policy: %w", err)
	}
	resolved, err := p.ResolveRole(*role)
	if err != nil {
		return err
	}
	model, ok := p.Models[resolved.Model]
	if !ok || !containsCapability(model.Capabilities, "chat") || !containsCapability(model.Capabilities, "generate") {
		return fmt.Errorf("Ollama role %q is not eligible for local chat inference", *role)
	}
	if err := policy.CheckLocalInferenceRights(h.getenv(), *role, resolved.Model); err != nil {
		return err
	}
	bytes := int64(math.Ceil(model.VRAMGBEstimate * float64(uint64(1)<<30)))
	if bytes <= 0 {
		return fmt.Errorf("Ollama policy has no positive VRAM estimate for %q", resolved.Model)
	}
	reservation, err := h.reservationStore().Create(*role, resolved.Model, bytes, time.Duration(*ttlSeconds)*time.Second)
	if err != nil {
		return err
	}
	reservation.ExpiresAt = reservation.ExpiresAt.UTC()
	if err := h.writeReceipt(*asJSON, reservation); err != nil {
		cleanupErr := h.reservationStore().Remove(reservation.ID)
		return errors.Join(fmt.Errorf("write Ollama model reservation receipt: %w", err), cleanupErr)
	}
	return nil
}

func (h *Handlers) ConfirmModelReservation(args []string) error {
	fs := flag.NewFlagSet("capacity confirm-model-reservation", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	id := fs.String("id", "", "Ollama model capacity reservation id")
	waitSeconds := fs.Int("wait-seconds", 30, "Maximum wait for Ollama's running capacity companion")
	asJSON := fs.Bool("json", false, "Emit a machine-readable reservation receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || strings.TrimSpace(*id) == "" || *waitSeconds < 0 || *waitSeconds > 300 {
		return errors.New("--id and --wait-seconds in [0, 300] are required; positional arguments are not accepted")
	}
	reservation, err := h.reservationStore().Read()
	if err != nil {
		return err
	}
	if reservation == nil || reservation.ID != strings.TrimSpace(*id) {
		return errors.New("Ollama model capacity reservation is missing, expired, or has a different ID")
	}
	waitDuration := time.Duration(*waitSeconds) * time.Second
	if waitDuration == 0 {
		waitDuration = time.Second
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), waitDuration)
	defer cancel()
	var claim capacityClaim
	var confirmed ModelReservation
	var releaseClaimOnStopped bool
	for {
		unlock, err := h.reservationStore().LockSync(waitCtx)
		if err != nil {
			return err
		}
		current, readErr := h.reservationStore().Read()
		if readErr != nil {
			err = readErr
		} else if current == nil || current.ID != strings.TrimSpace(*id) {
			err = errors.New("Ollama model capacity reservation changed or expired before capacity handoff")
		} else {
			// Reservation creation uses its own file lock. Revalidate the exact ID
			// after acquiring the companion lock so a queued confirmation cannot
			// admit capacity for an expired or replaced reservation.
			reservation = current
		}
		if err == nil {
			claim, err = h.ollamaClaim(waitCtx)
		}
		if err == nil || errors.Is(err, errReservationClaimNotReady) {
			var running []ensure.RunningModel
			running, err = h.client().ListRunning(waitCtx)
			if err != nil {
				stopped, statusErr := h.resourceStopped(waitCtx)
				if statusErr != nil {
					err = fmt.Errorf("inspect Ollama runtime after model-list failure (%v): %w", err, statusErr)
				} else if stopped {
					running = nil
					err = nil
				}
			}
			if err == nil {
				need, needErr := reservationCapacityNeed(*reservation, running)
				admissionBytes, admissionBytesErr := reservationAdmissionBytes(*reservation, running)
				if needErr != nil {
					err = needErr
				} else if admissionBytesErr != nil {
					err = admissionBytesErr
				}
				targetLoaded := targetModelResident(*reservation, running)
				if err == nil && targetLoaded && claim.AmountBytes >= need {
					// The complete target footprint is already resident and covered by
					// an active claim. Reuse it without charging that VRAM twice.
					admissionBytes = 0
				} else {
					observedResidentBytes := int64(0)
					if targetLoaded && admissionBytes <= 0 {
						// The target is already loaded; correct accounting only when the
						// broker independently attributes enough GPU process memory to
						// Ollama. This path cannot authorize another model load.
						observedResidentBytes = need
					} else if admissionBytes <= 0 {
						err = fmt.Errorf("Ollama's active claim (%d bytes) does not cover already resident models (%d bytes)", claim.AmountBytes, need)
					}
					var created bool
					var admittedClaimID string
					var admittedBytes int64
					var admissionErr error
					if err == nil {
						if _, err = h.reservationStore().BeginAdmission(*id); err == nil {
							admittedClaimID, admittedBytes, created, admissionErr = h.admitModelReservation(waitCtx, need, admissionBytes, observedResidentBytes)
						}
					}
					if admissionErr != nil {
						err = fmt.Errorf("admit Ollama model reservation: %w", admissionErr)
					} else if err == nil {
						releaseClaimOnStopped = created
						// Persist the admitted claim identity before any follow-up broker
						// read. OpenCode can then clean up a stopped runtime even if this
						// confirmation is canceled during verification.
						confirmed, err = h.reservationStore().Confirm(*id, admittedClaimID, admittedBytes, admissionBytes, releaseClaimOnStopped)
						if err != nil {
							err = fmt.Errorf("persist admitted Ollama model reservation: %w", err)
						}
					}
					if err == nil {
						claim, err = h.ollamaClaim(waitCtx)
						if err == nil && claim.AmountBytes < need {
							err = fmt.Errorf("%w: admitted claim has %d bytes, current models plus target need %d", errReservationClaimNotReady, claim.AmountBytes, need)
						}
					}
				}
				if err == nil {
					confirmed, err = h.reservationStore().Confirm(*id, claim.ClaimID, claim.AmountBytes, admissionBytes, releaseClaimOnStopped)
				}
			}
		}
		unlock()
		if err == nil {
			return h.writeReceipt(*asJSON, confirmed)
		}
		if !errors.Is(err, errReservationClaimNotReady) {
			return err
		}
		if *waitSeconds == 0 || waitCtx.Err() != nil {
			return fmt.Errorf("timed out waiting for Ollama's capacity companion to admit model %q: %w", reservation.Model, err)
		}
		select {
		case <-waitCtx.Done():
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (h *Handlers) resourceStopped(ctx context.Context) (bool, error) {
	exec := h.Exec
	if exec == nil {
		exec = companion.DefaultExec
	}
	out, err := exec(ctx, "vrooli", "resource", "status", resourceName, "--fast", "--json")
	if err != nil {
		return false, err
	}
	var status struct {
		Success bool   `json:"success"`
		Running *bool  `json:"running"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return false, fmt.Errorf("decode Ollama resource status: %w", err)
	}
	if !status.Success || status.Running == nil {
		return false, errors.New("Ollama resource status did not prove whether the runtime is running")
	}
	return !*status.Running, nil
}

func (h *Handlers) ReleaseModelReservation(args []string) error {
	fs := flag.NewFlagSet("capacity release-model-reservation", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	id := fs.String("id", "", "Ollama model capacity reservation id")
	asJSON := fs.Bool("json", false, "Emit a machine-readable release receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || strings.TrimSpace(*id) == "" {
		return errors.New("--id is required and positional arguments are not accepted")
	}
	unlock, err := h.reservationStore().LockSync(context.Background())
	if err != nil {
		return err
	}
	defer unlock()
	reservation, err := h.reservationStore().Read()
	if err != nil {
		return err
	}
	if reservation == nil || reservation.ID != strings.TrimSpace(*id) {
		return errors.New("Ollama model capacity reservation is missing, expired, or has a different ID")
	}
	running, err := h.client().ListRunning(context.Background())
	if err != nil {
		stopped, statusErr := h.resourceStopped(context.Background())
		if statusErr != nil {
			return fmt.Errorf("verify Ollama model residency before reservation handoff (%v); inspect managed runtime: %w", err, statusErr)
		}
		if !stopped {
			return fmt.Errorf("verify Ollama model residency before reservation handoff; managed runtime is running but its API is unavailable: %w", err)
		}
		running = nil
	}
	loaded := false
	var residentBytes int64
	for _, model := range running {
		residentBytes, err = addVRAM(residentBytes, model.SizeVRAM)
		if err != nil {
			return err
		}
		if sameModel(model.Name, reservation.Model) && model.SizeVRAM > 0 {
			loaded = true
		}
	}
	if residentBytes > 0 {
		claim, err := h.ollamaClaim(context.Background())
		if err != nil {
			return err
		}
		if claim.AmountBytes < residentBytes {
			return fmt.Errorf("Ollama resident claim (%d bytes) does not cover current model footprint (%d bytes); reservation retained", claim.AmountBytes, residentBytes)
		}
	} else if reservation.AdmissionPending {
		// The broker may have committed a replacement immediately before the
		// process stopped, but the exact claim ID was not durably returned. Do
		// not release a different active owner claim by inference. Its normal
		// lease/companion reconciliation bounds any leftover reservation.
	} else if reservation.ReleaseClaimOnStopped {
		if err := h.releaseStoppedReservationClaim(context.Background(), *reservation); err != nil {
			return fmt.Errorf("release unstarted Ollama model reservation claim: %w", err)
		}
	}
	if err := h.reservationStore().Remove(*id); err != nil {
		return err
	}
	return h.writeReceipt(*asJSON, map[string]any{"released": true, "model_loaded": loaded, "model": reservation.Model})
}

func (h *Handlers) releaseStoppedReservationClaim(ctx context.Context, reservation ModelReservation) error {
	if strings.TrimSpace(reservation.ClaimID) == "" {
		return errors.New("reservation has no owner-admitted claim ID")
	}
	exec := h.Exec
	if exec == nil {
		exec = companion.DefaultExec
	}
	out, err := exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return fmt.Errorf("read Ollama capacity claims before stopped-runtime cleanup: %w", err)
	}
	var result struct {
		Claims []capacityClaim `json:"claims"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("decode Ollama capacity claims before stopped-runtime cleanup: %w", err)
	}
	for _, claim := range result.Claims {
		if claim.ClaimID != reservation.ClaimID {
			continue
		}
		if claim.OwnerKind != "resource" || claim.OwnerID != resourceName || claim.ResourceKind != "vram" {
			return errors.New("reservation claim identity does not match Ollama's resource claim")
		}
		if _, err := exec(ctx, "vrooli", "capacity", "release", "--claim-id", reservation.ClaimID, "--json"); err != nil {
			return fmt.Errorf("release Ollama capacity claim %s: %w", reservation.ClaimID, err)
		}
		return nil
	}
	// An earlier stop or cleanup already terminated the exact reservation claim.
	return nil
}

func (h *Handlers) ollamaClaim(ctx context.Context) (capacityClaim, error) {
	ownerClaim, ok, err := h.ollamaOwnerClaim(ctx)
	if err != nil {
		return capacityClaim{}, err
	}
	reservation, err := h.reservationStore().Read()
	if err != nil {
		return capacityClaim{}, err
	}
	if reservation == nil {
		return capacityClaim{}, errors.New("Ollama model reservation expired before capacity handoff")
	}
	if ok {
		if ownerClaim.Status != "granted" || ownerClaim.ClaimID == "" {
			return capacityClaim{}, fmt.Errorf("%w: Ollama claim %s is %s", errReservationClaimNotReady, ownerClaim.ClaimID, ownerClaim.Status)
		}
		if ownerClaim.AmountBytes < reservation.RequestedBytes {
			return capacityClaim{}, fmt.Errorf("%w: granted claim (%d bytes) is below model reservation (%d bytes)", errReservationClaimNotReady, ownerClaim.AmountBytes, reservation.RequestedBytes)
		}
		return ownerClaim, nil
	}
	return capacityClaim{}, fmt.Errorf("%w: model %q needs %d bytes", errReservationClaimNotReady, reservation.Model, reservation.RequestedBytes)
}

func (h *Handlers) ollamaOwnerClaim(ctx context.Context) (capacityClaim, bool, error) {
	exec := h.Exec
	if exec == nil {
		exec = companion.DefaultExec
	}
	out, err := exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return capacityClaim{}, false, fmt.Errorf("read Ollama's broker capacity claim: %w", err)
	}
	var result struct {
		Claims []capacityClaim `json:"claims"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return capacityClaim{}, false, fmt.Errorf("decode Ollama broker capacity claims: %w", err)
	}
	var ownerClaim *capacityClaim
	for i := range result.Claims {
		claim := result.Claims[i]
		if claim.OwnerKind != "resource" || claim.OwnerID != resourceName || claim.ResourceKind != "vram" {
			continue
		}
		if ownerClaim != nil {
			return capacityClaim{}, false, errors.New("Ollama has multiple active VRAM claims; reservation handoff is ambiguous")
		}
		ownerClaim = &claim
	}
	if ownerClaim != nil {
		return *ownerClaim, true, nil
	}
	return capacityClaim{}, false, nil
}

func reservationCapacityNeed(reservation ModelReservation, running []ensure.RunningModel) (int64, error) {
	if reservation.RequestedBytes <= 0 {
		return 0, fmt.Errorf("Ollama model reservation must request positive bytes")
	}
	var resident int64
	var targetResident int64
	for _, model := range running {
		var err error
		resident, err = addVRAM(resident, model.SizeVRAM)
		if err != nil {
			return 0, err
		}
		if sameModel(model.Name, reservation.Model) && model.SizeVRAM > 0 {
			targetResident, err = addVRAM(targetResident, model.SizeVRAM)
			if err != nil {
				return 0, err
			}
		}
	}
	if targetResident > 0 {
		otherResident := resident - targetResident
		if reservation.RequestedBytes > targetResident {
			return addVRAM(otherResident, reservation.RequestedBytes)
		}
		return resident, nil
	}
	return addVRAM(resident, reservation.RequestedBytes)
}

func reservationAdmissionBytes(reservation ModelReservation, running []ensure.RunningModel) (int64, error) {
	var targetResident int64
	for _, model := range running {
		if sameModel(model.Name, reservation.Model) && model.SizeVRAM > 0 {
			var err error
			targetResident, err = addVRAM(targetResident, model.SizeVRAM)
			if err != nil {
				return 0, err
			}
		}
	}
	if targetResident >= reservation.RequestedBytes {
		return 0, nil
	}
	return reservation.RequestedBytes - targetResident, nil
}

func addVRAM(total, amount int64) (int64, error) {
	if total < 0 || amount < 0 {
		return 0, fmt.Errorf("Ollama model residency contains a negative VRAM value")
	}
	if amount > int64(^uint64(0)>>1)-total {
		return 0, fmt.Errorf("aggregate Ollama model VRAM overflows int64")
	}
	return total + amount, nil
}

func targetModelResident(reservation ModelReservation, running []ensure.RunningModel) bool {
	for _, model := range running {
		if sameModel(model.Name, reservation.Model) && model.SizeVRAM > 0 {
			return true
		}
	}
	return false
}

type capacityClaim struct {
	ClaimID      string `json:"claim_id"`
	OwnerKind    string `json:"owner_kind"`
	OwnerID      string `json:"owner_id"`
	ResourceKind string `json:"resource_kind"`
	Status       string `json:"status"`
	AmountBytes  int64  `json:"amount_bytes"`
}

func (h *Handlers) writeReceipt(asJSON bool, value any) error {
	if asJSON {
		return json.NewEncoder(h.Stdout).Encode(value)
	}
	_, err := fmt.Fprintf(h.Stdout, "%+v\n", value)
	return err
}

func (h *Handlers) getenv() func(string) string {
	if h.GetEnv != nil {
		return h.GetEnv
	}
	return os.Getenv
}

func containsCapability(capabilities []string, expected string) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}
