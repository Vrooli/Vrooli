// DOC: docs/reference/configuration.md#step-configuration — step config defaults table
package vps

import (
	"context"
	"errors"
	"log"
	"time"

	"scenario-to-cloud/ssh"
)

// StepConfig holds per-step execution parameters.
type StepConfig struct {
	CommandTimeout time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

// DefaultStepConfigs maps step IDs to their execution configuration.
// Steps not listed here inherit DefaultRunOptions() timeout (context-based).
var DefaultStepConfigs = map[string]StepConfig{
	"mkdir":             {CommandTimeout: 15 * time.Second},
	"bootstrap":         {CommandTimeout: 2 * time.Minute},
	"extract":           {CommandTimeout: 1 * time.Minute},
	"setup":             {CommandTimeout: 5 * time.Minute},
	"autoheal":          {CommandTimeout: 15 * time.Second},
	"verify_setup":      {CommandTimeout: 10 * time.Second},
	"scenario_stop":     {CommandTimeout: 30 * time.Second},
	"caddy_install":     {CommandTimeout: 1 * time.Minute},
	"caddy_config":      {CommandTimeout: 15 * time.Second},
	"firewall_inbound":  {CommandTimeout: 15 * time.Second},
	"secrets_provision": {CommandTimeout: 30 * time.Second},
	"resource_start":    {CommandTimeout: 2 * time.Minute, MaxRetries: 1, RetryDelay: 5 * time.Second},
	"scenario_deps":     {CommandTimeout: 2 * time.Minute},
	"scenario_target":   {CommandTimeout: 2 * time.Minute},
	"wait_for_ui":       {CommandTimeout: 45 * time.Second},
	"verify_local":      {CommandTimeout: 15 * time.Second},
	"verify_https":      {CommandTimeout: 20 * time.Second},
	"verify_origin":     {CommandTimeout: 20 * time.Second},
	"verify_public":     {CommandTimeout: 20 * time.Second},
}

// RunOptionsForStep returns RunOptions with the step-specific timeout merged in.
// Unknown steps get DefaultRunOptions() (no command timeout).
func RunOptionsForStep(stepID string) ssh.RunOptions {
	opts := ssh.DefaultRunOptions()
	if cfg, ok := DefaultStepConfigs[stepID]; ok {
		opts.CommandTimeout = cfg.CommandTimeout
	}
	return opts
}

// RunStepWithRetry executes an SSH command with per-step retry logic.
// If the step has MaxRetries > 0 and the error is retryable, it retries
// up to cfg.MaxRetries times with cfg.RetryDelay between attempts.
func RunStepWithRetry(
	ctx context.Context,
	runner ssh.Runner,
	sshCfg ssh.Config,
	stepID string,
	cmd string,
) error {
	cfg, ok := DefaultStepConfigs[stepID]
	if !ok {
		cfg = StepConfig{}
	}

	opts := RunOptionsForStep(stepID)
	var lastErr error

	attempts := 1 + cfg.MaxRetries
	for i := range attempts {
		_, err := runner.Run(ctx, sshCfg, cmd, opts)
		if err == nil {
			return nil
		}
		lastErr = err

		// Only retry if there are remaining attempts and error is retryable
		if i < attempts-1 {
			var sshErr *ssh.SSHError
			if errors.As(err, &sshErr) && sshErr.Retryable {
				log.Printf("step %s attempt %d/%d failed (retryable): %s", stepID, i+1, attempts, err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(cfg.RetryDelay):
				}
				continue
			}
		}
		break
	}
	return lastErr
}
