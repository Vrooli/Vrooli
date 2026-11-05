package logx

// Logger is the injectable logging seam for production packages that need to
// emit diagnostics without hard-wiring process-global logging.
type Logger interface {
	Errorf(format string, args ...any)
}
