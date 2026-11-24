//go:build audio_lab

package server

import (
	"audio-tools/internal/modules"
	"audio-tools/internal/testutil/mocks"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLabBuildMountsAllLabEndpoints(t *testing.T) {
	srv := ComposeForBuild(CompositionDeps{Logger: &mocks.FakeLogger{}})
	mounted := routeTemplates(t, srv)
	for _, endpoint := range modules.LabEndpoints() {
		require.Truef(t, routeServes(mounted, endpoint.Path), "lab composition omitted %s", endpoint.Path)
	}
}
