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
