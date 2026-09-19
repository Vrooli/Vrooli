package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
)

// Structural cause 5: the old loop observed SIGTERM only inside its ticker
// select, so a `vrooli scenario start` in flight kept it deaf until systemd's
// TimeoutStopSec killed it. One context now reaches the child, and the heal
// returns within cliinvoke's WaitDelay of the signal.
// [REQ:AUTOHEAL-P1-009] [REQ:INFRA-SHUTDOWN-001]
func TestSigtermDuringLifecycleCancelsChildWithinWaitDelay(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, `case "$1 $2" in
  "scenario start"|"scenario restart") echo "starting"; sleep 300; exit 0;;
esac
`+usageBody)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- heal(ctx, config, "test") }()
	time.Sleep(300 * time.Millisecond)
	cancel()
	signalled := time.Now()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("heal returned %v, want context.Canceled", err)
		}
		if elapsed := time.Since(signalled); elapsed > cliinvoke.DefaultWaitDelay+time.Second {
			t.Fatalf("heal took %v after cancellation, want within WaitDelay (%v)", elapsed, cliinvoke.DefaultWaitDelay)
		}
	case <-time.After(cliinvoke.DefaultWaitDelay + 5*time.Second):
		t.Fatal("heal did not return after cancellation; the child still holds it")
	}
	if countCalls(vrooliCalls(t, config.VrooliCmdPath), "scenario start") != 1 {
		t.Fatalf("calls = %v, want exactly one start", vrooliCalls(t, config.VrooliCmdPath))
	}
}

func TestEnsureAPIRunning(t *testing.T) {
	autoheal := fakeAPI(t, "Vrooli Autoheal API")
	foreign := fakeAPI(t, "mock-api")

	cases := []struct {
		name       string
		arrange    func(t *testing.T, config *Config)
		wantErr    bool
		wantClass  cliinvoke.Class
		wantHeal   bool
		wantPort   string
		wantStarts int
	}{
		{
			name: "already healthy: adopts without touching the lifecycle",
			arrange: func(t *testing.T, c *Config) {
				t.Setenv("API_PORT", autoheal)
				c.VrooliCmdPath = fakeVrooli(t, usageBody)
			},
			wantPort: autoheal,
		},
		{
			name: "foreign process on the announced port is not adopted; the scenario is started",
			arrange: func(t *testing.T, c *Config) {
				t.Setenv("API_PORT", foreign)
				c.VrooliCmdPath = fakeVrooli(t, startBody(c, autoheal)+usageBody)
			},
			wantPort:   autoheal,
			wantStarts: 1,
		},
		{
			name: "registry names a silent port: waits, then starts",
			arrange: func(t *testing.T, c *Config) {
				writeRegistryFile(t, c, "port", closedPort(t))
				c.VrooliCmdPath = fakeVrooli(t, startBody(c, autoheal)+usageBody)
			},
			wantPort:   autoheal,
			wantStarts: 1,
		},
		{
			name: "binary missing is non-healable",
			arrange: func(_ *testing.T, c *Config) {
				c.VrooliCmdPath = ""
				c.VrooliResolveErr = errors.New("vrooli binary not found; tried PATH")
			},
			wantErr:   true,
			wantClass: cliinvoke.BinaryMissing,
		},
		{
			name:       "usage error is non-healable",
			arrange:    func(t *testing.T, c *Config) { c.VrooliCmdPath = fakeVrooli(t, usageBody) },
			wantErr:    true,
			wantClass:  cliinvoke.Usage,
			wantStarts: 1,
		},
		{
			name: "lifecycle failure is healable",
			arrange: func(t *testing.T, c *Config) {
				c.VrooliCmdPath = fakeVrooli(t, "case \"$1 $2\" in \"scenario start\") echo 'build component api: exit status 1' >&2; exit 1;; esac\n"+usageBody)
			},
			wantErr:    true,
			wantClass:  cliinvoke.Lifecycle,
			wantHeal:   true,
			wantStarts: 1,
		},
		{
			name: "start succeeds but autoheal never answers: healable timeout",
			arrange: func(t *testing.T, c *Config) {
				c.VrooliCmdPath = fakeVrooli(t, startBody(c, foreign)+usageBody)
			},
			wantErr:    true,
			wantClass:  cliinvoke.Lifecycle,
			wantHeal:   true,
			wantStarts: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolatedHome(t)
			config := testConfig(t)
			tc.arrange(t, config)
			err := ensureAPIRunning(context.Background(), config, "test")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				var failure *healError
				if !errors.As(err, &failure) {
					t.Fatalf("err %v is not a healError", err)
				}
				if failure.Class != tc.wantClass || failure.Healable() != tc.wantHeal {
					t.Fatalf("class = %s healable = %v; want %s / %v", failure.Class, failure.Healable(), tc.wantClass, tc.wantHeal)
				}
			}
			if config.APIPort != tc.wantPort {
				t.Fatalf("APIPort = %q, want %q", config.APIPort, tc.wantPort)
			}
			if config.VrooliCmdPath != "" {
				if got := countCalls(vrooliCalls(t, config.VrooliCmdPath), "scenario start"); got != tc.wantStarts {
					t.Fatalf("scenario start invoked %d times, want %d: %v", got, tc.wantStarts, vrooliCalls(t, config.VrooliCmdPath))
				}
			}
		})
	}
}

func TestWaitForAPIHealthy(t *testing.T) {
	autoheal := fakeAPI(t, "Vrooli Autoheal API")
	foreign := fakeAPI(t, "mock-api")

	cases := []struct {
		name     string
		pending  string
		arrange  func(t *testing.T, config *Config)
		cancelIn time.Duration
		wantErr  error
		wantPort string
	}{
		{name: "pending autoheal port is adopted", pending: autoheal, wantPort: autoheal},
		{name: "pending foreign port is never adopted", pending: foreign, wantErr: errNotHealthy},
		{
			name:     "registry port appearing while waiting is adopted",
			arrange:  func(t *testing.T, c *Config) { writeRegistryFile(t, c, "port", autoheal) },
			wantPort: autoheal,
		},
		{name: "cancellation returns before the deadline", pending: foreign, cancelIn: 50 * time.Millisecond, wantErr: context.Canceled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolatedHome(t)
			config := testConfig(t)
			if tc.cancelIn > 0 {
				config.StartupTimeout = 10 * time.Second
			}
			if tc.arrange != nil {
				tc.arrange(t, config)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelIn > 0 {
				time.AfterFunc(tc.cancelIn, cancel)
			}
			started := time.Now()
			err := waitForAPIHealthy(ctx, config, tc.pending)
			switch {
			case tc.wantErr == nil && err != nil:
				t.Fatalf("err = %v", err)
			case tc.wantErr == errNotHealthy:
				var failure *healError
				if !errors.As(err, &failure) || failure.Class != cliinvoke.Lifecycle || !failure.Healable() {
					t.Fatalf("err = %v, want a healable lifecycle failure", err)
				}
			case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.cancelIn > 0 && time.Since(started) > time.Second {
				t.Fatalf("cancellation took %v", time.Since(started))
			}
			if config.APIPort != tc.wantPort {
				t.Fatalf("APIPort = %q, want %q", config.APIPort, tc.wantPort)
			}
		})
	}
}

// errNotHealthy marks the table rows that expect the healable deadline error.
var errNotHealthy = errors.New("not healthy")

// [REQ:AUTOHEAL-P0-014]
func TestRunLifecycleWithRecoveryDoesNotSpendBreakerSlotsOnUsageErrors(t *testing.T) {
	home := isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, usageBody)
	err := runLifecycleWithRecovery(context.Background(), config, "start")
	var failure *healError
	if !errors.As(err, &failure) || failure.Class != cliinvoke.Usage {
		t.Fatalf("err = %v, want usage", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".vrooli", "state", "vrooli-autoheal", "recovery-floor.json")); !os.IsNotExist(statErr) {
		t.Fatalf("recovery floor state was touched by a usage error: %v", statErr)
	}
}
