//go:build !linux

package orchestration

func (o *Orchestrator) StartAuthorizationBroker() (func() error, error) {
	return func() error { return nil }, nil
}
