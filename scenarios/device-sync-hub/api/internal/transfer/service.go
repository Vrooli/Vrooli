package transfer

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/blobstore"
)

// Tunable policy defaults. All are overridable via Config so tests pin exact
// values and operators can tighten limits per deployment.
const (
	// defaultRetention is applied when a create omits an explicit policy.
	defaultRetention = RetentionHeld
	// defaultHeldTTL is how long a Held item survives before auto-purge.
	defaultHeldTTL = 24 * time.Hour
	// defaultLiveTTL bounds an undelivered Live item so it still drains if no
	// device ever pulls it. Delivered Live items purge on the next sweep.
	defaultLiveTTL = 10 * time.Minute
	// defaultOwnerQuotaBytes caps total stored bytes per owner (0 = unlimited).
	defaultOwnerQuotaBytes int64 = 5 << 30 // 5 GiB
	// defaultDeviceQuotaBytes caps stored bytes contributed per origin device.
	defaultDeviceQuotaBytes int64 = 2 << 30 // 2 GiB
	// defaultMaxTextBytes caps an inline text snippet.
	defaultMaxTextBytes int64 = 1 << 20 // 1 MiB
)

// TrustChecker validates that a directed item's target is a trusted device of
// the same owner. Broadcast items skip it. Implemented at wiring by an adapter
// over the devices domain so transfer never imports devices' concrete types.
type TrustChecker interface {
	IsTrustedDevice(ctx context.Context, ownerID, deviceID string) (bool, error)
}

// Notifier receives owner-scoped item events so the realtime layer can push
// them to connected trusted devices. The production impl is the realtime hub;
// the default is a no-op so a scenario without realtime still functions.
type Notifier interface {
	// ItemArrived fires after an item is created (file or text).
	ItemArrived(ctx context.Context, item Item)
	// ItemDeleted fires after an item is deleted or purged.
	ItemDeleted(ctx context.Context, item Item)
}

// nopNotifier is the default when no realtime layer is wired.
type nopNotifier struct{}

func (nopNotifier) ItemArrived(context.Context, Item) {}
func (nopNotifier) ItemDeleted(context.Context, Item) {}

// Service is the application-layer surface the transfer handlers depend on. It
// owns validation, retention policy, quota enforcement, the delivery ACL, and
// the blob+event side effects of create/delete/purge. The handler is thin
// around it: authenticate the device, decode, call service, translate errors.
type Service interface {
	// CreateText stores a text snippet and returns the persisted item.
	CreateText(ctx context.Context, in CreateText) (Item, error)

	// CreateFile registers an already-stored file blob (the handler streamed the
	// bytes to the blob store first). On a metadata failure the caller deletes
	// the orphaned blob it created.
	CreateFile(ctx context.Context, in CreateFile) (Item, error)

	// CheckQuota reports whether accepting want more bytes from deviceID would
	// breach the owner or device quota, WITHOUT storing anything. The upload
	// handler calls it before streaming bytes so a doomed upload is rejected
	// early; CreateFile re-checks to close the race.
	CheckQuota(ctx context.Context, ownerID, deviceID string, want int64) error

	// List returns the items visible to deviceID of ownerID, newest first.
	List(ctx context.Context, ownerID, deviceID string, f ListFilter) ([]Item, error)

	// Get returns one item visible to deviceID, or ErrItemNotFound.
	Get(ctx context.Context, ownerID, deviceID, id string) (Item, error)

	// Delete removes an item (and its blobs) and emits ItemDeleted.
	Delete(ctx context.Context, ownerID, id string) (Item, error)

	// MarkDelivered flags a Live item delivered after a successful pull so the
	// next purge sweep removes it. Best-effort; a failure is logged, not fatal.
	MarkDelivered(ctx context.Context, ownerID, id string)

	// Purge runs one retention sweep: removes every due item (expired, or
	// delivered Live) and its blobs, emitting ItemDeleted for each. Returns the
	// number purged. The scheduler calls it on an interval.
	Purge(ctx context.Context) (int, error)
}

// Config configures NewService. Repo and Blobs are required; the rest default.
type Config struct {
	Repo  Repository
	Blobs blobstore.BlobStore
	Clock schedule.Clock
	Trust TrustChecker
	Notif Notifier
	Log   *log.Logger

	DefaultRetention Retention
	HeldTTL          time.Duration
	LiveTTL          time.Duration
	OwnerQuotaBytes  int64
	DeviceQuotaBytes int64
	MaxTextBytes     int64
}

type service struct {
	repo  Repository
	blobs blobstore.BlobStore
	clock schedule.Clock
	trust TrustChecker
	notif Notifier
	log   *log.Logger

	defRetention Retention
	heldTTL      time.Duration
	liveTTL      time.Duration
	ownerQuota   int64
	deviceQuota  int64
	maxText      int64
}

// NewService constructs the production Service, filling defaults for any
// optional dependency or limit left zero.
func NewService(cfg Config) Service {
	s := &service{
		repo:         cfg.Repo,
		blobs:        cfg.Blobs,
		clock:        cfg.Clock,
		trust:        cfg.Trust,
		notif:        cfg.Notif,
		log:          cfg.Log,
		defRetention: normalizeRetention(cfg.DefaultRetention, defaultRetention),
		heldTTL:      cfg.HeldTTL,
		liveTTL:      cfg.LiveTTL,
		ownerQuota:   cfg.OwnerQuotaBytes,
		deviceQuota:  cfg.DeviceQuotaBytes,
		maxText:      cfg.MaxTextBytes,
	}
	if s.clock == nil {
		s.clock = schedule.System()
	}
	if s.notif == nil {
		s.notif = nopNotifier{}
	}
	if s.log == nil {
		s.log = log.Default()
	}
	if s.heldTTL <= 0 {
		s.heldTTL = defaultHeldTTL
	}
	if s.liveTTL <= 0 {
		s.liveTTL = defaultLiveTTL
	}
	if cfg.OwnerQuotaBytes == 0 {
		s.ownerQuota = defaultOwnerQuotaBytes
	}
	if cfg.DeviceQuotaBytes == 0 {
		s.deviceQuota = defaultDeviceQuotaBytes
	}
	if s.maxText <= 0 {
		s.maxText = defaultMaxTextBytes
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) CreateText(ctx context.Context, in CreateText) (Item, error) {
	text := in.Text // preserve interior whitespace; only reject all-blank
	if strings.TrimSpace(text) == "" {
		return Item{}, ErrInvalidItem{Field: "text", Reason: "required"}
	}
	size := int64(len(text))
	if size > s.maxText {
		return Item{}, ErrInvalidItem{Field: "text", Reason: "snippet too large"}
	}
	if !utf8.ValidString(text) {
		return Item{}, ErrInvalidItem{Field: "text", Reason: "must be valid UTF-8"}
	}
	if err := s.validateTarget(ctx, in.OwnerID, in.TargetDeviceID); err != nil {
		return Item{}, err
	}
	if err := s.CheckQuota(ctx, in.OwnerID, in.OriginDeviceID, size); err != nil {
		return Item{}, err
	}

	retention := normalizeRetention(in.Retention, s.defRetention)
	now := s.clock.Now().UTC()
	item := Item{
		OwnerID:        in.OwnerID,
		OriginDeviceID: in.OriginDeviceID,
		Kind:           KindText,
		Name:           strings.TrimSpace(in.Name),
		MIME:           "text/plain; charset=utf-8",
		SizeBytes:      size,
		Text:           text,
		Retention:      retention,
		TargetDeviceID: strings.TrimSpace(in.TargetDeviceID),
		ExpiresAt:      s.expiryFor(retention, now),
		CreatedAt:      now,
	}
	stored, err := s.repo.Create(ctx, item)
	if err != nil {
		return Item{}, err
	}
	s.notif.ItemArrived(ctx, stored)
	return stored, nil
}

func (s *service) CreateFile(ctx context.Context, in CreateFile) (Item, error) {
	if strings.TrimSpace(in.BlobKey) == "" {
		return Item{}, ErrInvalidItem{Field: "blob", Reason: "missing stored blob"}
	}
	if in.SizeBytes <= 0 {
		return Item{}, ErrInvalidItem{Field: "file", Reason: "empty file"}
	}
	if err := s.validateTarget(ctx, in.OwnerID, in.TargetDeviceID); err != nil {
		return Item{}, err
	}
	if err := s.CheckQuota(ctx, in.OwnerID, in.OriginDeviceID, in.SizeBytes); err != nil {
		return Item{}, err
	}

	retention := normalizeRetention(in.Retention, s.defRetention)
	now := s.clock.Now().UTC()
	mime := strings.TrimSpace(in.MIME)
	if mime == "" {
		mime = "application/octet-stream"
	}
	item := Item{
		OwnerID:        in.OwnerID,
		OriginDeviceID: in.OriginDeviceID,
		Kind:           KindFile,
		Name:           strings.TrimSpace(in.Name),
		MIME:           mime,
		SizeBytes:      in.SizeBytes,
		BlobKey:        in.BlobKey,
		ThumbKey:       strings.TrimSpace(in.ThumbKey),
		Retention:      retention,
		TargetDeviceID: strings.TrimSpace(in.TargetDeviceID),
		ExpiresAt:      s.expiryFor(retention, now),
		CreatedAt:      now,
	}
	stored, err := s.repo.Create(ctx, item)
	if err != nil {
		return Item{}, err
	}
	s.notif.ItemArrived(ctx, stored)
	return stored, nil
}

func (s *service) CheckQuota(ctx context.Context, ownerID, deviceID string, want int64) error {
	if want <= 0 {
		return nil
	}
	if s.ownerQuota > 0 {
		used, err := s.repo.UsageByOwner(ctx, ownerID)
		if err != nil {
			return err
		}
		if used+want > s.ownerQuota {
			return ErrQuotaExceeded{Scope: "owner", LimitByte: s.ownerQuota, UsedByte: used, WantByte: want}
		}
	}
	if s.deviceQuota > 0 && deviceID != "" {
		used, err := s.repo.UsageByDevice(ctx, ownerID, deviceID)
		if err != nil {
			return err
		}
		if used+want > s.deviceQuota {
			return ErrQuotaExceeded{Scope: "device", LimitByte: s.deviceQuota, UsedByte: used, WantByte: want}
		}
	}
	return nil
}

func (s *service) List(ctx context.Context, ownerID, deviceID string, f ListFilter) ([]Item, error) {
	return s.repo.ListVisible(ctx, ownerID, deviceID, f)
}

func (s *service) Get(ctx context.Context, ownerID, deviceID, id string) (Item, error) {
	return s.repo.GetVisible(ctx, ownerID, deviceID, id)
}

func (s *service) Delete(ctx context.Context, ownerID, id string) (Item, error) {
	deleted, err := s.repo.Delete(ctx, ownerID, id)
	if err != nil {
		return Item{}, err
	}
	s.deleteBlobs(ctx, deleted)
	s.notif.ItemDeleted(ctx, deleted)
	return deleted, nil
}

func (s *service) MarkDelivered(ctx context.Context, ownerID, id string) {
	if err := s.repo.MarkDelivered(ctx, ownerID, id); err != nil {
		s.log.Printf("transfer.MarkDelivered(%q): %v", id, err)
	}
}

func (s *service) Purge(ctx context.Context) (int, error) {
	due, err := s.repo.DueForPurge(ctx, s.clock.Now().UTC())
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, item := range due {
		if err := s.repo.PurgeByID(ctx, item.ID); err != nil {
			s.log.Printf("transfer.Purge(%q): %v", item.ID, err)
			continue
		}
		s.deleteBlobs(ctx, item)
		s.notif.ItemDeleted(ctx, item)
		purged++
	}
	return purged, nil
}

// validateTarget enforces that a directed item names a trusted device of the
// same owner. Broadcast (empty target) is always allowed. A nil TrustChecker
// (test wiring) accepts any non-empty target.
func (s *service) validateTarget(ctx context.Context, ownerID, targetDeviceID string) error {
	targetDeviceID = strings.TrimSpace(targetDeviceID)
	if targetDeviceID == "" {
		return nil
	}
	if s.trust == nil {
		return nil
	}
	ok, err := s.trust.IsTrustedDevice(ctx, ownerID, targetDeviceID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTarget{DeviceID: targetDeviceID}
	}
	return nil
}

// deleteBlobs best-effort removes an item's stored bytes and thumbnail. Storage
// errors are logged, not surfaced — the metadata row is already gone, so a
// stranded blob is reclaimable garbage, never a correctness problem.
func (s *service) deleteBlobs(ctx context.Context, item Item) {
	if s.blobs == nil {
		return
	}
	if item.BlobKey != "" {
		if err := s.blobs.Delete(ctx, item.BlobKey); err != nil {
			s.log.Printf("transfer: delete blob %q: %v", item.BlobKey, err)
		}
	}
	if item.ThumbKey != "" {
		if err := s.blobs.Delete(ctx, item.ThumbKey); err != nil {
			s.log.Printf("transfer: delete thumb %q: %v", item.ThumbKey, err)
		}
	}
}

// expiryFor stamps the auto-purge time for a retention policy: Pinned never
// expires (zero time), Live drains after a short bound, Held after its window.
func (s *service) expiryFor(r Retention, now time.Time) time.Time {
	switch r {
	case RetentionLive:
		return now.Add(s.liveTTL)
	case RetentionHeld:
		return now.Add(s.heldTTL)
	default: // Pinned
		return time.Time{}
	}
}
