package orchestrator

import "io"

// Option configures an Orchestrator.
type Option func(*Orchestrator)

// WithLogger sets the logger for status messages.
func WithLogger(w io.Writer) Option {
	return func(o *Orchestrator) {
		o.logger = w
	}
}

// WithPreflightChecker sets the preflight checker implementation.
func WithPreflightChecker(p PreflightChecker) Option {
	return func(o *Orchestrator) {
		o.preflight = p
	}
}

// WithHealthChecker sets the health checker implementation for auto-recovery.
// If set, this enables automatic browserless recovery on health failures.
func WithHealthChecker(h HealthChecker) Option {
	return func(o *Orchestrator) {
		o.healthChecker = h
		// HealthChecker also implements PreflightChecker
		if o.preflight == nil {
			o.preflight = h
		}
	}
}

// WithAutoRecovery enables automatic browserless recovery.
func WithAutoRecovery(enabled bool) Option {
	return func(o *Orchestrator) {
		o.autoRecoveryEnabled = enabled
	}
}

// WithAutoRecoveryOptions sets the auto-recovery configuration.
func WithAutoRecoveryOptions(opts AutoRecoveryOptions) Option {
	return func(o *Orchestrator) {
		o.autoRecoveryOpts = opts
		o.autoRecoveryEnabled = true
	}
}

// WithBrowserClient sets the browser client implementation.
func WithBrowserClient(b BrowserClient) Option {
	return func(o *Orchestrator) {
		o.browser = b
	}
}

// WithArtifactWriter sets the artifact writer implementation.
func WithArtifactWriter(a ArtifactWriter) Option {
	return func(o *Orchestrator) {
		o.artifacts = a
	}
}

// WithHandshakeDetector sets the handshake detector implementation.
func WithHandshakeDetector(h HandshakeDetector) Option {
	return func(o *Orchestrator) {
		o.handshake = h
	}
}

// WithPayloadGenerator sets the payload generator implementation.
func WithPayloadGenerator(p PayloadGenerator) Option {
	return func(o *Orchestrator) {
		o.payloadGen = p
	}
}

// DefaultAutoRecoveryOptions returns the standard auto-recovery options.
func DefaultAutoRecoveryOptions() AutoRecoveryOptions {
	return AutoRecoveryOptions{
		SharedMode:    false,
		MaxRetries:    1,
		ContainerName: "vrooli-browserless",
	}
}

// WithScenarioStarter sets the scenario starter implementation for auto-start.
func WithScenarioStarter(s ScenarioStarter) Option {
	return func(o *Orchestrator) {
		o.scenarioStarter = s
	}
}
