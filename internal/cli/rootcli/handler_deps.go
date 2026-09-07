package rootcli

import (
	"context"
	"io"
	"time"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	packageapp "github.com/vrooli/vrooli/internal/app/package"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
)

// HandlerDeps is the shared transport-to-handler dependency shell. Domain
// handlers use only the fields they need; domain-specific services retain
// their distinct types rather than being erased behind interface{}.
type HandlerDeps[C any] struct {
	Stdout           func(C) io.Writer
	Stderr           func(C) io.Writer
	Root             func(C) string
	Globals          func(C) GlobalOptions
	OutputFormat     func(C) (cliout.Format, error)
	OperationContext func(C) context.Context
	Stdin            func(C) io.Reader
	HomeDir          func(C) (string, error)
	EnsureCLI        func(C, string) error

	Probes             func(C) []authapp.SignInProbe
	ServiceFor         func(C) capacityapp.Service
	Service            func(C) contractapp.Service
	Validate           func(C) (contractapp.ValidationOutput, error)
	Store              func(C) (*baselinefloor.Store, error)
	Clock              func() time.Time
	ResourceController func(C) (*resources.Controller, error)

	PackageScenarioOperations func(C) (packageapp.ScenarioRuntime, error)
	PackageLifecycleRunner    func(C) (packageapp.ScenarioPhaseRunner, error)
	TestGenieRunner           func(C, string, io.Writer, io.Writer) error

	ScenarioOperations      func(C) (scenarioapp.ScenarioOperations, error)
	TimingStore             func(C) (*scenarioruntime.SQLiteStore, error)
	LifecycleRunner         func(C) (scenarioapp.PhaseRunner, error)
	EnvValidator            func(C) (scenarioapp.EnvironmentValidator, error)
	OpenURL                 func(C, string) error
	LaunchDetached          func(C, ...string) error
	RunSubprocess           func(C, shell.Spec) error
	RemoteScenarioCall      func(C, string, string, string, []string, bool) ([]byte, error)
	LocateTestGenieCLI      func(C) (string, error)
	LocateBusinessHealthCLI func(C) (string, error)
	LocateCompleteCLI       func(C) (string, error)
	CommandEnv              func(C) []string

	ResolveRoot func(C) (string, error)
	Version     func(C) string
}

// ResolveOperationContext returns the caller-owned operation context when a
// transport provides one and preserves the legacy background behavior for
// direct handler tests and compatibility callers.
func ResolveOperationContext[C any](deps HandlerDeps[C], ctx C) context.Context {
	if deps.OperationContext != nil {
		if operationCtx := deps.OperationContext(ctx); operationCtx != nil {
			return operationCtx
		}
	}
	return context.Background()
}
