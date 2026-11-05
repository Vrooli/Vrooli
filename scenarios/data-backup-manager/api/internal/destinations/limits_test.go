package destinations_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/engine"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

// TestLimits_UsageState drives GetDestinationUsage with seeded RepoStats and
// asserts the WITHIN / NEAR / OVER thresholds behave correctly.
func TestLimits_UsageState(t *testing.T) {
	ctx := context.Background()
	const cap = int64(1000)

	cases := []struct {
		name       string
		usageBytes int64
		wantState  destinations.UsageState
	}{
		{"comfortably within", 500, destinations.UsageStateWithin},
		{"exactly 89%", 890, destinations.UsageStateWithin},
		{"at 90% (near)", 900, destinations.UsageStateNear},
		{"99% (near)", 990, destinations.UsageStateNear},
		{"exactly at cap (over)", 1000, destinations.UsageStateOver},
		{"over cap (over)", 1200, destinations.UsageStateOver},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			usage := tc.usageBytes // capture
			eng := &enginemocks.FakeKopiaEngine{
				RepoStatsFn: func(_ context.Context, _ string) (engine.RepoStats, error) {
					return engine.RepoStats{SizeBytes: usage}, nil
				},
			}
			repo := mocks.NewFakeRepository()
			svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

			d, err := svc.CreateDestination(ctx, destinations.CreateInput{
				Name:      fmt.Sprintf("cap-dest-%d", i),
				Backend:   destinations.BackendFilesystem,
				Location:  "/mnt/cap",
				CapBytes:  cap,
				CapPolicy: destinations.CapPolicyAlertBlock,
			})
			if err != nil {
				t.Fatalf("create: %v", err)
			}

			report, err := svc.GetDestinationUsage(ctx, d.ID)
			if err != nil {
				t.Fatalf("GetDestinationUsage: %v", err)
			}
			if report.UsageState != tc.wantState {
				t.Fatalf("UsageState = %q, want %q (usage=%d cap=%d)",
					report.UsageState, tc.wantState, usage, cap)
			}
			if report.UsageBytes != usage {
				t.Fatalf("UsageBytes = %d, want %d", report.UsageBytes, usage)
			}
		})
	}

	t.Run("no cap always WITHIN", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{
			RepoStatsFn: func(_ context.Context, _ string) (engine.RepoStats, error) {
				return engine.RepoStats{SizeBytes: 999999}, nil
			},
		}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "nocap-dest",
			Backend:  destinations.BackendFilesystem,
			Location: "/mnt/nocap",
			CapBytes: 0,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		report, err := svc.GetDestinationUsage(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetDestinationUsage: %v", err)
		}
		if report.UsageState != destinations.UsageStateWithin {
			t.Fatalf("no-cap dest should always be WITHIN, got %q", report.UsageState)
		}
	})
}

// TestLimits_DefaultAlertBlock drives WouldBlock and proves:
//  1. With default ALERT_BLOCK policy and cap_bytes > 0, WouldBlock returns
//     blocked=true when usage + pending exceeds the cap.
//  2. The engine fake's Calls log contains NO delete call; WouldBlock only
//     inspects RepoStats.
//  3. ALERT_ONLY policy never blocks even when over cap.
func TestLimits_DefaultAlertBlock(t *testing.T) {
	ctx := context.Background()
	const cap = int64(1000)
	const currentUsage = int64(900)

	t.Run("ALERT_BLOCK blocks when over cap", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{
			RepoStatsFn: func(_ context.Context, _ string) (engine.RepoStats, error) {
				return engine.RepoStats{SizeBytes: currentUsage}, nil
			},
		}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:      "block-dest",
			Backend:   destinations.BackendFilesystem,
			Location:  "/mnt/block",
			CapBytes:  cap,
			CapPolicy: destinations.CapPolicyAlertBlock,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		// pending = 200; usage(900) + pending(200) = 1100 > cap(1000) → block.
		blocked, reason, err := svc.WouldBlock(ctx, d.ID, 200)
		if err != nil {
			t.Fatalf("WouldBlock: %v", err)
		}
		if !blocked {
			t.Fatal("expected blocked=true when usage+pending exceeds cap")
		}
		if reason == "" {
			t.Fatal("expected non-empty reason when blocked")
		}

		// Assert the call log contains RepoStats but no delete-like call.
		for _, call := range eng.Calls {
			if strings.HasPrefix(call, "RepoDelete") || strings.HasPrefix(call, "SnapshotDelete") {
				t.Fatalf("unexpected delete call in engine log: %q", call)
			}
		}
		hasStats := false
		for _, call := range eng.Calls {
			if len(call) >= 9 && call[:9] == "RepoStats" {
				hasStats = true
				break
			}
		}
		if !hasStats {
			t.Fatalf("expected RepoStats in engine calls; calls = %v", eng.Calls)
		}
	})

	t.Run("ALERT_BLOCK does not block when within cap", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{
			RepoStatsFn: func(_ context.Context, _ string) (engine.RepoStats, error) {
				return engine.RepoStats{SizeBytes: 100}, nil
			},
		}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:      "within-dest",
			Backend:   destinations.BackendFilesystem,
			Location:  "/mnt/within",
			CapBytes:  cap,
			CapPolicy: destinations.CapPolicyAlertBlock,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		blocked, _, err := svc.WouldBlock(ctx, d.ID, 50)
		if err != nil {
			t.Fatalf("WouldBlock: %v", err)
		}
		if blocked {
			t.Fatal("expected blocked=false when within cap")
		}
	})

	t.Run("ALERT_ONLY never blocks", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{
			RepoStatsFn: func(_ context.Context, _ string) (engine.RepoStats, error) {
				// Way over cap.
				return engine.RepoStats{SizeBytes: 9999}, nil
			},
		}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:      "alertonly-dest",
			Backend:   destinations.BackendFilesystem,
			Location:  "/mnt/alertonly",
			CapBytes:  cap,
			CapPolicy: destinations.CapPolicyAlertOnly,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		blocked, _, err := svc.WouldBlock(ctx, d.ID, 9999)
		if err != nil {
			t.Fatalf("WouldBlock: %v", err)
		}
		if blocked {
			t.Fatal("ALERT_ONLY policy must never block, even when over cap")
		}
	})

	t.Run("no cap never blocks", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "nocap-block-dest",
			Backend:  destinations.BackendFilesystem,
			Location: "/mnt/nocap2",
			CapBytes: 0,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		blocked, _, err := svc.WouldBlock(ctx, d.ID, 1<<50)
		if err != nil {
			t.Fatalf("WouldBlock: %v", err)
		}
		if blocked {
			t.Fatal("cap_bytes=0 (no cap) must never block")
		}
	})
}
