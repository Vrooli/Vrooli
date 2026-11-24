// Package provision coordinates intent, credit and provider operations.
package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"compute-manager/internal/clock"
	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	"compute-manager/internal/provider/akash"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
)

type Service struct {
	Intents  intent.Service
	Meter    meter.Service
	Prices   PriceCatalog
	MoneyCap MoneyCap
	// Effects defaults to stub when absent. Every production source must have
	// an explicit live setting before provision may cross its mutation boundary.
	Effects   *effectgate.Gate
	Provider  provider.Provider
	Providers *provider.Registry
	Window    time.Duration
	DB        interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	Now func() time.Time
	Mu  *sync.Mutex
}

// RenewReservations advances each held reservation before releasing its
// predecessor. A transient release failure therefore leaves two holds rather
// than leaving a running instance unreserved.
func (s Service) RenewReservations(ctx context.Context) error {
	if s.DB == nil || s.Window <= 0 {
		return nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,i.tenant,i.reservation_id,r.quantity FROM instances i JOIN reservations r ON r.id=i.reservation_id WHERE i.state='running' AND r.state='held'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type candidate struct {
		instanceID, tenant, predecessor string
		amount                          int64
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.instanceID, &item.tenant, &item.predecessor, &item.amount); err != nil {
			return err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range candidates {
		amount := float64(item.amount)
		if amount < 1 {
			amount = 1
		}
		next, err := s.Meter.Reserve(ctx, item.tenant, amount, s.Window)
		if err != nil {
			return fmt.Errorf("renew reservation for %s: %w", item.instanceID, err)
		}
		now := clock.System{}.Now
		if s.Now != nil {
			now = s.Now
		}
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO reservations (id,instance_id,supersedes,meter_key,state,held_at,quantity) VALUES (?,?,?,?,?,?,?)`, next.ID, item.instanceID, item.predecessor, "compute_minutes", "held", now().UTC().Format(time.RFC3339Nano), int64(amount)); err != nil {
			_ = s.Meter.Release(ctx, next.ID)
			return fmt.Errorf("record renewed reservation for %s: %w", item.instanceID, err)
		}
		if err := s.Meter.Release(ctx, item.predecessor); err != nil {
			return fmt.Errorf("release predecessor reservation for %s: %w", item.instanceID, err)
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=? AND state='held'`, now().UTC().Format(time.RFC3339Nano), item.predecessor); err != nil {
			return fmt.Errorf("close predecessor reservation for %s: %w", item.instanceID, err)
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE instances SET reservation_id=? WHERE id=?`, next.ID, item.instanceID); err != nil {
			return fmt.Errorf("link renewed reservation for %s: %w", item.instanceID, err)
		}
	}
	return nil
}

func (s Service) ReleaseReservation(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return s.Meter.Release(ctx, id)
}

func (s Service) FinalizeReservation(ctx context.Context, id string, amount float64) error {
	if id == "" {
		return nil
	}
	return s.Meter.Finalize(ctx, id, amount)
}

func (s Service) Create(ctx context.Context, req intent.Request, amount float64) (intent.Record, provider.Instance, error) {
	return s.create(ctx, req, amount, s.Provider)
}

func (s Service) CreateWithProvider(ctx context.Context, req intent.Request, amount float64, selected provider.Provider) (intent.Record, provider.Instance, error) {
	return s.create(ctx, req, amount, selected)
}

func (s Service) create(ctx context.Context, req intent.Request, amount float64, selected provider.Provider) (intent.Record, provider.Instance, error) {
	if selected == nil {
		return intent.Record{}, provider.Instance{}, fmt.Errorf("provider is unavailable")
	}
	if req.Provider != "" && req.Provider != selected.Name() {
		return intent.Record{}, provider.Instance{}, fmt.Errorf("requested provider %q does not match selected provider %q", req.Provider, selected.Name())
	}
	req.Provider = selected.Name()
	// CD3's admitted selection always includes one IPv4 address; carry that
	// exact selection through both the quote and provider request.
	req.Spec.IPv4 = true
	if s.Mu != nil {
		s.Mu.Lock()
		defer s.Mu.Unlock()
	}
	if req.IdempotencyKey == "" {
		return intent.Record{}, provider.Instance{}, fmt.Errorf("idempotency key is required")
	}
	if existing, lookupErr := s.Intents.Store.GetByIDOrKey(ctx, req.IdempotencyKey); lookupErr == nil {
		return existing, provider.Instance{}, fmt.Errorf("existing intent %q in state %q requires reconciliation; provider create was not repeated", req.IdempotencyKey, existing.State)
	} else if !errors.Is(lookupErr, intent.ErrNotFound) {
		return intent.Record{}, provider.Instance{}, fmt.Errorf("check idempotency key before spend admission: %w", lookupErr)
	}
	record, err := s.Intents.CreateIntent(ctx, req)
	if err != nil {
		return intent.Record{}, provider.Instance{}, err
	}
	if record.State != intent.StateOpen {
		return record, provider.Instance{}, nil
	}
	if s.DB == nil {
		return record, provider.Instance{}, fmt.Errorf("spend admission unavailable: durable quote store is not configured")
	}
	var quote StoredQuote
	if selected.Name() == "local-qemu" {
		err = validateLocalQEMUSelection(req.Spec.Region, req.Spec.Size, req.Spec.Profile)
	}
	if err == nil {
		quote, err = priceSelection(ctx, s.Prices, selected.Name(), req.Spec.Region, req.Spec.Size, req.IdempotencyKey, req.Spec.IPv4, s.MoneyCap, s.now())
	}
	if err == nil && selected.Name() == akash.Name {
		err = akash.ValidateIntentBinding(quote.PriceProvenanceJSON, record.Spec)
	}
	if err != nil {
		unavailable := StoredQuote{IdempotencyKey: req.IdempotencyKey, Provider: selected.Name(), Region: req.Spec.Region, Size: req.Spec.Size, IPv4: true, Term: "monthly", State: "refused_billing_facts", Reason: err.Error(), SourceVersion: "unavailable", CreatedAt: s.now()}
		if storeErr := persistQuoteUnavailable(ctx, s.DB, &unavailable, s.MoneyCap); storeErr != nil {
			err = errors.Join(err, storeErr)
		}
		record.ResolvedAt = s.now().UTC()
		refusalErr := err
		if transitionErr := s.Intents.Transition(ctx, &record, intent.StateRefused); transitionErr != nil {
			refusalErr = errors.Join(refusalErr, fmt.Errorf("record refused intent: %w", transitionErr))
		}
		return record, provider.Instance{}, refusalErr
	}
	if s.Effects == nil || !s.Effects.AllowsMutation(selected.Name()) {
		effect := effectgate.Stub
		if s.Effects != nil {
			effect = s.Effects.ModeFor(selected.Name())
		}
		quote.State = "refused_effect_gate"
		quote.Reason = fmt.Sprintf("source effect gate is %q; provider call is disabled", effect)
	}
	newQuote, err := persistQuote(ctx, s.DB, &quote, s.MoneyCap)
	if err != nil {
		return record, provider.Instance{}, err
	}
	if !newQuote {
		return record, provider.Instance{}, fmt.Errorf("existing stored quote for idempotency key %q requires reconciliation; provider create was not repeated", req.IdempotencyKey)
	}
	if quote.State != "approved" {
		record.ResolvedAt = s.now().UTC()
		refusalErr := fmt.Errorf("spend refused: %s", quote.Reason)
		if transitionErr := s.Intents.Transition(ctx, &record, intent.StateRefused); transitionErr != nil {
			refusalErr = errors.Join(refusalErr, fmt.Errorf("record refused intent: %w", transitionErr))
		}
		return record, provider.Instance{}, refusalErr
	}
	ceilingHold, err := s.Meter.AcquireCeilingHold(ctx, req.RequestedBy, amount, s.Window)
	if err != nil {
		record.ResolvedAt = clock.System{}.Now().UTC()
		if transitionErr := s.Intents.Transition(ctx, &record, intent.StateRefused); transitionErr != nil {
			err = errors.Join(err, fmt.Errorf("record refused intent: %w", transitionErr))
		}
		return record, provider.Instance{}, err
	}
	defer func() { _ = s.Meter.ReleaseCeilingHold(ctx, ceilingHold) }()
	reservation, err := s.Meter.Reserve(ctx, req.RequestedBy, amount, s.Window)
	if err != nil {
		record.ResolvedAt = clock.System{}.Now().UTC()
		refusalErr := fmt.Errorf("capacity refused: %w", err)
		if transitionErr := s.Intents.Transition(ctx, &record, intent.StateRefused); transitionErr != nil {
			refusalErr = errors.Join(refusalErr, fmt.Errorf("record refused intent: %w", transitionErr))
		}
		return record, provider.Instance{}, refusalErr
	}
	record.ReservationID = reservation.ID
	if s.DB != nil {
		now := time.Now
		if s.Now != nil {
			now = s.Now
		}
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO reservations (id,intent_id,meter_key,state,held_at,quantity) VALUES (?,?,?,?,?,?)`, reservation.ID, record.ID, "compute_minutes", "held", now().UTC().Format(time.RFC3339Nano), int64(amount)); err != nil {
			_ = s.Meter.Release(ctx, reservation.ID)
			return record, provider.Instance{}, fmt.Errorf("record reservation: %w", err)
		}
	}
	if err := s.Intents.SaveOpen(ctx, record); err != nil {
		_ = s.Meter.Release(ctx, reservation.ID)
		if s.DB != nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=?`, clock.System{}.Now().UTC().Format(time.RFC3339Nano), reservation.ID)
		}
		return record, provider.Instance{}, err
	}
	req.Spec.Tags = withIntentTag(req.Spec.Tags, record.ID)
	created, err := selected.Create(ctx, req.Spec)
	if err != nil {
		if errors.Is(err, provider.ErrCreateResponseLost) {
			return record, provider.Instance{}, err
		}
		_ = s.Meter.Release(ctx, reservation.ID)
		if s.DB != nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET state='released',settled_at=? WHERE id=?`, clock.System{}.Now().UTC().Format(time.RFC3339Nano), reservation.ID)
		}
		record.ResolvedAt = time.Now().UTC()
		if transitionErr := s.Intents.Transition(ctx, &record, intent.StateRefused); transitionErr != nil {
			err = errors.Join(err, fmt.Errorf("record refused intent: %w", transitionErr))
		}
		return record, provider.Instance{}, err
	}
	record.InstanceID = created.ID
	record.ResolvedAt = clock.System{}.Now().UTC()
	if err := s.Intents.Transition(ctx, &record, intent.StateFulfilled); err != nil {
		return record, created, err
	}
	if s.DB != nil {
		_, _ = s.DB.ExecContext(ctx, `UPDATE reservations SET instance_id=? WHERE id=?`, created.ID, reservation.ID)
	}
	return record, created, nil
}

func validateLocalQEMUSelection(region, size, profile string) error {
	if region != "local" {
		return fmt.Errorf("local-qemu requires exact host-local region 'local'")
	}
	if profile != "headless-linux" && profile != "desktop-linux" {
		return fmt.Errorf("local-qemu profile %q is unsupported or missing", profile)
	}
	if size != profile {
		return fmt.Errorf("local-qemu quote size %q must exactly match profile %q", size, profile)
	}
	return nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func withIntentTag(tags map[string]string, intentID string) map[string]string {
	result := make(map[string]string, len(tags)+1)
	for key, value := range tags {
		result[key] = value
	}
	result["vrooli-intent-id"] = intentID
	return result
}
