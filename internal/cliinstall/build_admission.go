package cliinstall

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	publiccapacity "github.com/vrooli/vrooli/packages/capacity"
)

// cliBuildMemoryReservation is the existing declared Go-module builder
// estimate. It is provisional for CLI compilation until an owner measures it.
const cliBuildMemoryReservation = int64(394 << 20)
const cliBuildCleanupTimeout = 5 * time.Second
const cliBuildStopGrace = 2 * time.Second
const cliBuildGroupPollInterval = 25 * time.Millisecond

var errCLIProcessGroupCleanupUnverified = errors.New("CLI build process-group cleanup is unverified")

type cliAdmissionBroker interface {
	Acquire(context.Context, string, int64, int64) (publiccapacity.Lease, publiccapacity.Verdict, error)
	Close() error
}

var (
	openCLIBuildBroker = func(ctx context.Context) (cliAdmissionBroker, error) {
		return publiccapacity.NewBroker(ctx, "")
	}
	cliBuildHeartbeatInterval = publiccapacity.RecommendedLeaseHeartbeatInterval
)

func withCLIInstallAdmission(ctx context.Context, name string, work func(context.Context) error) (resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(name) == "" || work == nil {
		return fmt.Errorf("CLI build admission requires an installable name and work")
	}
	broker, err := openCLIBuildBroker(ctx)
	if err != nil {
		return fmt.Errorf("CLI build admission unavailable: open capacity broker: %w", err)
	}
	defer func() {
		if closeErr := broker.Close(); closeErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close CLI build capacity broker: %w", closeErr))
		}
	}()

	ownerID, err := cliBuildOwnerID(name)
	if err != nil {
		return err
	}
	lease, verdict, err := broker.Acquire(ctx, ownerID, cliBuildMemoryReservation, 0)
	if err != nil {
		if lease != nil {
			resultErr = errors.Join(err, releaseCLILease(ctx, lease))
			return fmt.Errorf("CLI build admission failed: %w", resultErr)
		}
		return fmt.Errorf("CLI build admission failed: %w", err)
	}
	if verdict.Kind != internalcapacity.VerdictGrant || lease == nil {
		if lease != nil {
			if releaseErr := releaseCLILease(ctx, lease); releaseErr != nil {
				return fmt.Errorf("CLI build admission %s (%s); partial lease release failed: %w", verdict.Kind, verdict.Reason, releaseErr)
			}
		}
		return fmt.Errorf("CLI build admission %s (%s); retry when host capacity is available", verdict.Kind, verdict.Reason)
	}
	renewable, ok := lease.(publiccapacity.RenewableLease)
	if !ok {
		if releaseErr := releaseCLILease(ctx, lease); releaseErr != nil {
			return fmt.Errorf("CLI build received a non-renewable lease and release failed: %w", releaseErr)
		}
		return fmt.Errorf("CLI build received a non-renewable capacity lease; refusing an unrenewed build")
	}

	workCtx, cancelWork := context.WithCancel(ctx)
	stopRenewal := make(chan struct{})
	renewalDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(cliBuildHeartbeatInterval())
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
				heartbeatCtx, cancel := context.WithTimeout(context.WithoutCancel(workCtx), cliBuildCleanupTimeout)
				heartbeatErr := renewable.Heartbeat(heartbeatCtx)
				cancel()
				if heartbeatErr != nil {
					cancelWork()
					renewalDone <- fmt.Errorf("CLI build capacity lease renewal failed: %w", heartbeatErr)
					return
				}
			}
		}
	}()

	resultErr = errors.Join(work(workCtx), ctx.Err())
	close(stopRenewal)
	cancelWork()
	resultErr = errors.Join(resultErr, <-renewalDone)
	if errors.Is(resultErr, errCLIProcessGroupCleanupUnverified) {
		return errors.Join(resultErr, errors.New("capacity lease was retained because build descendants may still be active"))
	}
	cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), cliBuildCleanupTimeout)
	if releaseErr := lease.Release(cleanupCtx); releaseErr != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("release CLI build capacity lease: %w", releaseErr))
	}
	cancelCleanup()
	return resultErr
}

func releaseCLILease(ctx context.Context, lease publiccapacity.Lease) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cliBuildCleanupTimeout)
	defer cancel()
	return lease.Release(cleanupCtx)
}

func cliBuildOwnerID(name string) (string, error) {
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("create CLI build owner identity: %w", err)
	}
	name = strings.NewReplacer(":", "-", "/", "-", "\\", "-").Replace(strings.TrimSpace(name))
	ownerID := publiccapacity.OwnerIDFor("cli-install", fmt.Sprintf("pid-%d", os.Getpid()), name+"-"+hex.EncodeToString(nonce[:]))
	if ownerID == "" {
		return "", fmt.Errorf("invalid CLI build owner identity")
	}
	return ownerID, nil
}
