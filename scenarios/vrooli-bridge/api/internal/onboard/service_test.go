package onboard_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"

	"github.com/stretchr/testify/require"
)

const (
	testNodeID   = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
	testPassword = "s3cr3t-owner-password"
	testCode     = "PAIRINGCODEABCDEF0123456789ABCDEF"
)

// bootstrapSteps is the ordered step vocabulary the bootstrap script emits.
var bootstrapSteps = []string{
	"detect-os", "prereqs", "clone", "setup", "build-agent", "build-cli",
	"node-key", "pair-redeem", "pin-verify", "service-install", "autostart", "verify-online",
}

// successMarkers builds the canonical full-success marker stream (run envelope +
// every step ok), with the node id carried in the pair-redeem / run-ok details.
func successMarkers(nodeID string) []onboard.Marker {
	ms := []onboard.Marker{{Event: "run-start", Detail: "vrooli-bridge node bootstrap"}}
	for _, step := range bootstrapSteps {
		ms = append(ms, onboard.Marker{Event: "step-start", Step: step})
		detail := ""
		switch step {
		case "pair-redeem":
			detail = "paired as " + nodeID
		case "pin-verify":
			detail = "pinned key present, node " + nodeID
		}
		ms = append(ms, onboard.Marker{Event: "step-ok", Step: step, Detail: detail})
	}
	ms = append(ms, onboard.Marker{Event: "run-ok", Detail: "node " + nodeID + " paired and online"})
	return ms
}

func newTestService(repo *mocks.FakeRepository, driver *mocks.FakeSSHDriver, issuer *mocks.FakeCodeIssuer, confirmer *mocks.FakeOnlineConfirmer) onboard.Service {
	return onboard.NewService(repo, driver, issuer, confirmer, clock.System{})
}

func validInput() onboard.StartInput {
	return onboard.StartInput{
		Actor:           "owner-1",
		Host:            "web-01.example.com",
		User:            "deploy",
		Password:        []byte(testPassword),
		NodeName:        "web-01",
		TargetRevision:  "a1b2c3d",
		ControlPlaneURL: "https://cp.example.com",
	}
}

// waitTerminal drives the block-once Wait to a terminal op.
func waitTerminal(t *testing.T, svc onboard.Service, id string) onboard.Op {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, timedOut, err := svc.Wait(ctx, id, 5*time.Second)
	require.NoError(t, err)
	require.False(t, timedOut, "op did not reach a terminal state in time")
	require.True(t, op.State.Terminal())
	return op
}

func TestStart_SuccessFullFlow(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	issuer := &mocks.FakeCodeIssuer{Code: testCode}
	confirmer := &mocks.FakeOnlineConfirmer{Online: true}
	svc := newTestService(repo, driver, issuer, confirmer)

	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)
	require.NotEmpty(t, dec.OpID)
	require.False(t, dec.DryRun)

	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, onboard.StateSucceeded, op.State)
	require.Equal(t, testNodeID, op.NodeID)
	require.Equal(t, int32(0), op.ExitCode)
	require.Empty(t, op.FailureReason)

	// The pairing code reached the bootstrap over the injection seam (not argv).
	require.Equal(t, testCode, string(driver.PairingCode()))
	require.NotContains(t, driver.CapturedArgs, testCode, "pairing code must never appear in bootstrap argv")
	require.Contains(t, driver.CapturedArgs, "--control-plane-url")
	require.Contains(t, driver.CapturedArgs, "https://cp.example.com")
	require.Contains(t, driver.CapturedArgs, "--node-name")
	require.Equal(t, "web-01", issuer.LastParams.NodeName)
	require.Equal(t, testNodeID, confirmer.LastID)

	// The persisted step history covers the orchestrator phases AND every emitted
	// bootstrap step, in order.
	_, events, err := svc.GetOp(context.Background(), dec.OpID)
	require.NoError(t, err)
	stepsSeen := map[string]bool{}
	for _, ev := range events {
		stepsSeen[ev.StepID] = true
	}
	require.True(t, stepsSeen[onboard.StepSSHSetup])
	require.True(t, stepsSeen[onboard.StepPushScript])
	require.True(t, stepsSeen[onboard.StepVerifyOnline])
	require.True(t, stepsSeen[onboard.StepRun])
	for _, step := range bootstrapSteps {
		require.True(t, stepsSeen[step], "missing persisted step %q", step)
	}
	// Sequences are strictly increasing.
	for i := 1; i < len(events); i++ {
		require.Greater(t, events[i].Sequence, events[i-1].Sequence)
	}
}

func TestStart_SecretsNeverPersisted(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	issuer := &mocks.FakeCodeIssuer{Code: testCode}
	confirmer := &mocks.FakeOnlineConfirmer{Online: true}
	svc := newTestService(repo, driver, issuer, confirmer)

	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)
	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, onboard.StateSucceeded, op.State)

	// Neither the password nor the pairing code may appear anywhere in the durable
	// op row.
	opBlob := op.Host + op.User + op.NodeName + op.TargetRevision + op.RepoURL + op.NodeID + string(op.FailureReason)
	require.NotContains(t, opBlob, testPassword)
	require.NotContains(t, opBlob, testCode)

	// …nor in any persisted step-event detail.
	_, events, err := svc.GetOp(context.Background(), dec.OpID)
	require.NoError(t, err)
	for _, ev := range events {
		require.NotContains(t, ev.Detail, testPassword, "step %q leaked the password", ev.StepID)
		require.NotContains(t, ev.Detail, testCode, "step %q leaked the pairing code", ev.StepID)
	}
}

func TestStart_DryRunTouchesNothing(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{}
	svc := newTestService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true})

	in := validInput()
	in.DryRun = true
	dec, err := svc.Start(context.Background(), in)
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.OpID)

	// No op created, no host touched, password zeroed.
	ops, err := svc.ListOps(context.Background(), onboard.ListFilter{})
	require.NoError(t, err)
	require.Empty(t, ops)
	require.Equal(t, 0, driver.FirstTouchCalls)
	require.Equal(t, strings.Repeat("\x00", len(testPassword)), string(in.Password))
}

func TestStart_Validation(t *testing.T) {
	svc := newTestService(mocks.NewFakeRepository(), &mocks.FakeSSHDriver{}, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{})

	cases := map[string]func(*onboard.StartInput){
		"host":              func(in *onboard.StartInput) { in.Host = "" },
		"control_plane_url": func(in *onboard.StartInput) { in.ControlPlaneURL = "" },
		"target_revision":   func(in *onboard.StartInput) { in.TargetRevision = "" },
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			in := validInput()
			mutate(&in)
			_, err := svc.Start(context.Background(), in)
			var invalid onboard.ErrInvalid
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, field, invalid.Field)
		})
	}
}

func TestStart_FailAtEachPhase(t *testing.T) {
	t.Run("first-touch", func(t *testing.T) {
		driver := &mocks.FakeSSHDriver{FirstTouchErr: errors.New("auth failed")}
		svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{})
		dec, err := svc.Start(context.Background(), validInput())
		require.NoError(t, err)
		op := waitTerminal(t, svc, dec.OpID)
		require.Equal(t, onboard.StateFailed, op.State)
		require.Equal(t, onboard.FailureSSHSetup, op.FailureReason)
	})

	t.Run("push-script", func(t *testing.T) {
		driver := &mocks.FakeSSHDriver{PushScriptErr: errors.New("scp failed")}
		svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{})
		dec, err := svc.Start(context.Background(), validInput())
		require.NoError(t, err)
		op := waitTerminal(t, svc, dec.OpID)
		require.Equal(t, onboard.StateFailed, op.State)
		require.Equal(t, onboard.FailureScriptPush, op.FailureReason)
	})

	t.Run("code-issue", func(t *testing.T) {
		driver := &mocks.FakeSSHDriver{}
		issuer := &mocks.FakeCodeIssuer{Err: errors.New("mint failed")}
		svc := newTestService(mocks.NewFakeRepository(), driver, issuer, &mocks.FakeOnlineConfirmer{})
		dec, err := svc.Start(context.Background(), validInput())
		require.NoError(t, err)
		op := waitTerminal(t, svc, dec.OpID)
		require.Equal(t, onboard.StateFailed, op.State)
		require.Equal(t, onboard.FailurePairingIssue, op.FailureReason)
	})

	t.Run("verify-offline", func(t *testing.T) {
		driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
		confirmer := &mocks.FakeOnlineConfirmer{Online: false}
		svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, confirmer)
		dec, err := svc.Start(context.Background(), validInput())
		require.NoError(t, err)
		op := waitTerminal(t, svc, dec.OpID)
		require.Equal(t, onboard.StateFailed, op.State)
		require.Equal(t, onboard.FailureVerifyOnline, op.FailureReason)
	})

	t.Run("bootstrap-ok-but-no-node-id", func(t *testing.T) {
		// A run-ok with no node id anywhere: the orchestrator cannot verify.
		markers := []onboard.Marker{{Event: "run-start"}, {Event: "run-ok", Detail: "done"}}
		driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: markers}
		svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true})
		dec, err := svc.Start(context.Background(), validInput())
		require.NoError(t, err)
		op := waitTerminal(t, svc, dec.OpID)
		require.Equal(t, onboard.StateFailed, op.State)
		require.Equal(t, onboard.FailureVerifyOnline, op.FailureReason)
	})
}

func TestStart_BootstrapExitCodes(t *testing.T) {
	cases := []struct {
		exit   int
		reason onboard.FailureReason
	}{
		{2, onboard.FailureBootstrapUsage},
		{3, onboard.FailureUnsupportedPlatform},
		{4, onboard.FailurePairing},
		{1, onboard.FailureBootstrap},
		{7, onboard.FailureBootstrap},
	}
	for _, tc := range cases {
		t.Run(string(tc.reason), func(t *testing.T) {
			// run-fail markers up to a point, then a non-zero exit.
			markers := []onboard.Marker{{Event: "run-start"}, {Event: "step-start", Step: "clone"}, {Event: "step-fail", Step: "clone", Detail: "boom"}, {Event: "run-fail", Detail: "boom"}}
			driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: markers, RunBootstrapExit: tc.exit}
			svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true})
			dec, err := svc.Start(context.Background(), validInput())
			require.NoError(t, err)
			op := waitTerminal(t, svc, dec.OpID)
			require.Equal(t, onboard.StateFailed, op.State)
			require.Equal(t, tc.reason, op.FailureReason)
			require.Equal(t, int32(tc.exit), op.ExitCode)
		})
	}
}

func TestCancel_DrivesToCancelled(t *testing.T) {
	// RunBootstrap blocks until the op-scoped context is cancelled.
	driver := &mocks.FakeSSHDriver{RunBootstrapBlock: true}
	svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true})

	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)

	// Wait until the orchestrator is in the bootstrapping phase, then cancel.
	require.Eventually(t, func() bool {
		op, _, err := svc.GetOp(context.Background(), dec.OpID)
		return err == nil && op.State == onboard.StateBootstrapping
	}, 3*time.Second, 10*time.Millisecond)

	_, err = svc.Cancel(context.Background(), dec.OpID)
	require.NoError(t, err)

	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, onboard.StateCancelled, op.State)
}

func TestResumeInterrupted_MarksOrphansFailed(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := newTestService(repo, &mocks.FakeSSHDriver{}, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{})

	// Seed an op left mid-flight by a "previous process".
	repo.Seed(onboard.Op{ID: "orphan-1", Host: "h", State: onboard.StateBootstrapping, CreatedAt: time.Now().UTC()})
	repo.Seed(onboard.Op{ID: "done-1", Host: "h", State: onboard.StateSucceeded, CreatedAt: time.Now().UTC()})

	n, err := svc.ResumeInterrupted(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, n)

	orphan, _, err := svc.GetOp(context.Background(), "orphan-1")
	require.NoError(t, err)
	require.Equal(t, onboard.StateFailed, orphan.State)
	require.Equal(t, onboard.FailureInterrupted, orphan.FailureReason)
	require.False(t, orphan.FinishedAt.IsZero())

	done, _, err := svc.GetOp(context.Background(), "done-1")
	require.NoError(t, err)
	require.Equal(t, onboard.StateSucceeded, done.State)
}

func TestWait_TimesOutWhileRunning(t *testing.T) {
	driver := &mocks.FakeSSHDriver{RunBootstrapBlock: true}
	svc := newTestService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true})
	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)

	op, timedOut, err := svc.Wait(context.Background(), dec.OpID, 100*time.Millisecond)
	require.NoError(t, err)
	require.True(t, timedOut)
	require.False(t, op.State.Terminal())

	// Clean up the blocked goroutine.
	_, _ = svc.Cancel(context.Background(), dec.OpID)
	waitTerminal(t, svc, dec.OpID)
}
