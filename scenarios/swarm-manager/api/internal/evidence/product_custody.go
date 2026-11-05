package evidence

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/freshness-go/treedigest"
)

var errProductCustodyUnavailable = errors.New("exclusive product custody is not installed")
var errProductFenceRefused = errors.New("product publication fence refused")

// productCustody is a protocol over storage whose ALL writers are controlled
// by its installed owner. There is deliberately no production constructor:
// current shared checkout custody and producer ci:v1 evidence are unqualified.
// A mutex alone never proves custody. Only disposable tests instantiate this
// type; no DevelopmentEvidenceReader or live startup path installs it.
//
// The owner freezes the complete named input closure, configuration and
// toolchain. Capture uses the existing strict BuildInputManifest algorithm,
// never HEAD, td: aliasing or a bespoke digest. Mutation/publication are the
// only permitted operations once real backing-store custody is established.
type productCustody struct {
	mu       sync.Mutex
	capture  func(context.Context) (treedigest.InputManifest, error)
	epoch    uint64
	identity string
	deadline time.Time
	poisoned bool
}
type productPublicationFence struct {
	mu       sync.Mutex
	owner    *productCustody
	epoch    uint64
	identity string
	active   bool
}

func (f *productPublicationFence) check(ctx context.Context) error {
	if f == nil {
		return errProductFenceRefused
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.active || f.owner == nil || f.owner.poisoned || f.owner.epoch != f.epoch || f.owner.identity != f.identity || ctx.Err() != nil || !time.Now().Before(f.owner.deadline) {
		return errProductFenceRefused
	}
	return nil
}

// publish excludes owned mutations until the callback has returned from its
// canonical rename AND directory flush. The callback is synchronous: it must
// never defer publication into another goroutine. The callback MUST use check immediately
// before publication; deadline expiry never releases custody during a save.
// Errors after rename are unknown durability, not guaranteed zero effects.
func (o *productCustody) publish(ctx context.Context, epoch uint64, identity string, commit func(check func() error) error) error {
	if o == nil || o.capture == nil {
		return errProductCustodyUnavailable
	}
	if commit == nil || !o.mu.TryLock() {
		return errProductFenceRefused
	}
	defer o.mu.Unlock()
	f := &productPublicationFence{owner: o, epoch: epoch, identity: identity, active: true}
	defer func() { f.mu.Lock(); f.active = false; f.mu.Unlock() }()
	if f.check(ctx) != nil {
		return errProductFenceRefused
	}
	manifest, err := o.capture(ctx)
	if err != nil || manifest.SchemaVersion != treedigest.InputManifestSchemaVersion || manifest.Identity != identity {
		return errProductFenceRefused
	}
	if f.check(ctx) != nil {
		return errProductFenceRefused
	}
	var checked atomic.Bool
	err = commit(func() error {
		if e := f.check(ctx); e != nil {
			return e
		}
		checked.Store(true)
		return nil
	})
	if err != nil {
		return err
	}
	if !checked.Load() {
		return errProductFenceRefused
	}
	return nil
}

// mutate is a private installed-owner operation, never an HTTP callback or a
// credential/grant. The storage owner alone constructs mutation closures. A
// failed mutation/capture poisons custody because bytes may already have changed;
// it cannot be advertised as a no-effect rejection or retried at the old epoch.
func (o *productCustody) mutate(ctx context.Context, expected uint64, apply func() error) error {
	if o == nil || o.capture == nil {
		return errProductCustodyUnavailable
	}
	if apply == nil || !o.mu.TryLock() {
		return errProductFenceRefused
	}
	defer o.mu.Unlock()
	if o.poisoned || o.epoch != expected || expected == ^uint64(0) || ctx.Err() != nil || !time.Now().Before(o.deadline) {
		return errProductFenceRefused
	}
	prior, err := o.capture(ctx)
	if err != nil || prior.Identity != o.identity {
		o.poisoned = true
		return errProductFenceRefused
	}
	if err = apply(); err != nil {
		o.poisoned = true
		return err
	}
	next, err := o.capture(ctx)
	if err != nil || next.SchemaVersion != treedigest.InputManifestSchemaVersion {
		o.poisoned = true
		return errProductFenceRefused
	}
	o.epoch++
	o.identity = next.Identity
	return nil
}
