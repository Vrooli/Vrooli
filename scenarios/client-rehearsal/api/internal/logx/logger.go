// Package logx defines the narrow logging seam shared by API components.
package logx

import "log"

// Logger is the output surface used by the API. Production wires log.Default
// at the composition root; tests can provide a recording or discard logger.
// seam: logx.Logger substitutes package-wide logging for API behavior.
type Logger interface {
	Printf(format string, v ...any)
	Fatalf(format string, v ...any)
}

var _ Logger = (*log.Logger)(nil)
