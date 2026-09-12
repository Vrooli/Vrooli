// DOC: docs/reference/configuration.md#step-configuration — step config defaults table
package vps

import "time"

// StepConfig holds per-step execution parameters.
type StepConfig struct {
	CommandTimeout time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

// DefaultStepConfigs maps plan action ids to their execution configuration.
// Actions not listed here inherit DefaultRunOptions() timeout (context-based).
// "install_vrooli" is the native CLI install sub-step run inside release.activate.
var DefaultStepConfigs = map[string]StepConfig{
	"host.prepare":               {CommandTimeout: 2 * time.Minute},
	"edge.firewall.allow":        {CommandTimeout: 15 * time.Second},
	"data.inventory":             {CommandTimeout: 30 * time.Second},
	"release.verify":             {CommandTimeout: 1 * time.Minute},
	"release.stage":              {CommandTimeout: 2 * time.Minute},
	"release.activate":           {CommandTimeout: 2 * time.Minute},
	"install_vrooli":             {CommandTimeout: 2 * time.Minute},
	"config.apply":               {CommandTimeout: 5 * time.Minute},
	"workload.stop":              {CommandTimeout: 30 * time.Second},
	"edge.route.apply":           {CommandTimeout: 1 * time.Minute},
	"credentials.provision":      {CommandTimeout: 30 * time.Second},
	"runtime.start_dependencies": {CommandTimeout: 2 * time.Minute, MaxRetries: 1, RetryDelay: 5 * time.Second},
	"workload.start":             {CommandTimeout: 2 * time.Minute},
	"verify.readiness":           {CommandTimeout: 20 * time.Second},
	"release.retain_predecessor": {CommandTimeout: 15 * time.Second},
}
