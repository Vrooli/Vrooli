package runtimeapp

// Service owns typed runtime supervisor and recovery operations. It carries no
// CLI parsing.
type Service struct {
	Version       string
	ResolveRootFn func() (string, error)
}
