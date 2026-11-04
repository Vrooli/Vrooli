package lifecycle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	publiccapacity "github.com/vrooli/vrooli/packages/capacity"
)

const buildAdmissionCleanupTimeout = 5 * time.Second

type buildAdmissionBroker interface {
	Acquire(context.Context, string, int64, int64) (publiccapacity.Lease, publiccapacity.Verdict, error)
	Close() error
}

type buildAdmissionBrokerFactory func(context.Context, string) (buildAdmissionBroker, error)

type buildAdmissionSession struct {
	once    sync.Once
	factory buildAdmissionBrokerFactory
	home    string
	broker  buildAdmissionBroker
	err     error
}

func (r *Runner) newBuildAdmissionSession() *buildAdmissionSession {
	factory := r.buildAdmissionFactory
	if factory == nil {
		factory = productionBuildAdmissionBroker
	}
	return &buildAdmissionSession{factory: factory, home: r.Home}
}

func (s *buildAdmissionSession) get(ctx context.Context) (buildAdmissionBroker, error) {
	s.once.Do(func() {
		s.broker, s.err = s.factory(ctx, s.home)
		if s.err == nil && s.broker == nil {
			s.err = fmt.Errorf("capacity broker factory returned nil")
		}
	})
	return s.broker, s.err
}

func (s *buildAdmissionSession) close() error {
	if s == nil || s.broker == nil {
		return nil
	}
	return s.broker.Close()
}

func productionBuildAdmissionBroker(ctx context.Context, home string) (buildAdmissionBroker, error) {
	path, err := internalcapacity.DefaultDBPath(home)
	if err != nil {
		return nil, fmt.Errorf("resolve capacity ledger: %w", err)
	}
	broker, err := publiccapacity.NewBroker(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("open capacity broker: %w", err)
	}
	return broker, nil
}

// withBuildAdmission holds one broker operation claim around the complete
// install/build/publish sequence for a stale component. The estimate is the
// builder's declared reservation, not a measured process peak. A missing
// estimate or unavailable/denied broker fails closed rather than running an
// unclaimed build.
func (r *Runner) withBuildAdmission(ctx context.Context, itemSlug, component, runID string, reservation int64, work func(context.Context) (int, error)) (count int, resultErr error) {
	session := r.newBuildAdmissionSession()
	count, resultErr = r.withBuildAdmissionSessionAtInterval(ctx, session, itemSlug, component, runID, reservation, publiccapacity.RecommendedLeaseHeartbeatInterval(), work)
	if closeErr := session.close(); closeErr != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("close component %s capacity broker: %w", component, closeErr))
	}
	return count, resultErr
}

func (r *Runner) withBuildAdmissionAtInterval(ctx context.Context, itemSlug, component, runID string, reservation int64, heartbeatInterval time.Duration, work func(context.Context) (int, error)) (count int, resultErr error) {
	session := r.newBuildAdmissionSession()
	count, resultErr = r.withBuildAdmissionSessionAtInterval(ctx, session, itemSlug, component, runID, reservation, heartbeatInterval, work)
	if closeErr := session.close(); closeErr != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("close component %s capacity broker: %w", component, closeErr))
	}
	return count, resultErr
}

func (r *Runner) withBuildAdmissionSessionAtInterval(ctx context.Context, session *buildAdmissionSession, itemSlug, component, runID string, reservation int64, heartbeatInterval time.Duration, work func(context.Context) (int, error)) (count int, resultErr error) {
	if reservation <= 0 {
		return 0, fmt.Errorf("component %s has no declared build memory reservation; retry after its builder declares an estimate", component)
	}
	if heartbeatInterval <= 0 {
		return 0, fmt.Errorf("component %s has invalid capacity lease heartbeat interval", component)
	}
	broker, err := session.get(ctx)
	if err != nil {
		return 0, fmt.Errorf("component %s build admission unavailable: %w", component, err)
	}
	if broker == nil {
		return 0, fmt.Errorf("component %s build admission unavailable: broker factory returned nil", component)
	}
	ownerID, err := buildAdmissionOwnerID(itemSlug, runID, component)
	if err != nil {
		return 0, err
	}
	lease, verdict, err := broker.Acquire(ctx, ownerID, reservation, 0)
	if err != nil {
		if lease != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), buildAdmissionCleanupTimeout)
			releaseErr := lease.Release(cleanupCtx)
			cancel()
			if releaseErr != nil {
				err = errors.Join(err, fmt.Errorf("release partial component %s capacity lease: %w", component, releaseErr))
			}
		}
		return 0, fmt.Errorf("component %s build admission failed: %w", component, err)
	}
	if verdict.Kind != internalcapacity.VerdictGrant || lease == nil {
		if lease != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), buildAdmissionCleanupTimeout)
			releaseErr := lease.Release(cleanupCtx)
			cancel()
			if releaseErr != nil {
				return 0, fmt.Errorf("component %s build admission %s (%s); releasing partial lease: %w", component, verdict.Kind, verdict.Reason, releaseErr)
			}
		}
		return 0, fmt.Errorf("component %s build admission %s (%s); retry when capacity is available", component, verdict.Kind, verdict.Reason)
	}
	renewable, ok := lease.(publiccapacity.RenewableLease)
	if !ok {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), buildAdmissionCleanupTimeout)
		releaseErr := lease.Release(cleanupCtx)
		cancel()
		if releaseErr != nil {
			return 0, fmt.Errorf("component %s received a non-renewable capacity lease and release failed: %w", component, releaseErr)
		}
		return 0, fmt.Errorf("component %s received a non-renewable capacity lease; refusing long-running build", component)
	}

	workCtx, cancelWork := context.WithCancel(ctx)
	stopRenewal := make(chan struct{})
	renewalDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopRenewal:
				renewalDone <- nil
				return
			case <-workCtx.Done():
				renewalDone <- nil
				return
			case <-ticker.C:
				heartbeatCtx, cancel := context.WithTimeout(context.WithoutCancel(workCtx), buildAdmissionCleanupTimeout)
				heartbeatErr := renewable.Heartbeat(heartbeatCtx)
				cancel()
				if heartbeatErr != nil {
					cancelWork()
					renewalDone <- fmt.Errorf("component %s build admission lease renewal failed: %w", component, heartbeatErr)
					return
				}
			}
		}
	}()

	count, workErr := work(workCtx)
	workErr = errors.Join(workErr, ctx.Err())
	close(stopRenewal)
	cancelWork()
	renewalErr := <-renewalDone
	cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), buildAdmissionCleanupTimeout)
	releaseErr := lease.Release(cleanupCtx)
	cancelCleanup()
	joined := errors.Join(workErr, renewalErr)
	if releaseErr != nil {
		joined = errors.Join(joined, fmt.Errorf("release component %s build capacity lease: %w", component, releaseErr))
	}
	if joined != nil {
		return count, joined
	}
	return count, nil
}

func buildAdmissionOwnerID(itemSlug, runID, component string) (string, error) {
	if strings.Contains(itemSlug, ":") || strings.Contains(component, ":") {
		return "", fmt.Errorf("invalid component build admission identity")
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("create component build admission identity: %w", err)
	}
	runID = strings.TrimSpace(runID)
	if runID == "" || strings.Contains(runID, ":") {
		runID = fmt.Sprintf("pid-%d-%s", os.Getpid(), hex.EncodeToString(nonce[:]))
	}
	owner := publiccapacity.OwnerIDFor(itemSlug, runID, "build-"+component)
	if owner == "" {
		return "", fmt.Errorf("invalid component build admission identity")
	}
	return owner, nil
}
