package observability

import (
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// stubProxy returns canned payloads / errors for every Proxy method.
type stubProxy struct {
	snapshot map[string]any
	err      error
}

func (s *stubProxy) FetchObservability(context.Context, string, bool) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityRefresh(context.Context) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityDiagnostics(context.Context, map[string]any) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilitySessions(context.Context) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityCleanup(context.Context) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityMetrics(context.Context) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityPipelineTest(context.Context, map[string]any) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) FetchObservabilityConfigRuntime(context.Context) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) UpdateObservabilityConfig(context.Context, string, string) (map[string]any, error) {
	return s.snapshot, s.err
}

func (s *stubProxy) ResetObservabilityConfig(context.Context, string) (map[string]any, error) {
	return s.snapshot, s.err
}

func discardLog() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestModule_BuildsMount(t *testing.T) {
	mount := Module(Deps{
		Proxy:  &stubProxy{},
		Logger: discardLog(),
	})
	assert.NotEmpty(t, mount.Path)
	assert.NotNil(t, mount.Handler)
}

func TestModule_PanicsWithoutLogger(t *testing.T) {
	assert.Panics(t, func() {
		Module(Deps{Proxy: &stubProxy{}})
	})
}

func TestModule_PanicsWithoutProxy(t *testing.T) {
	assert.Panics(t, func() {
		Module(Deps{Logger: discardLog()})
	})
}
