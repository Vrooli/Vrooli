// Package readiness exposes the shared Experience Manager readiness profile
// reader to BAS consumers over Connect-RPC.
package readiness

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness/readinessconnect"

	"github.com/vrooli/browser-automation-studio/services/readiness"
)

var ProtoFile = readinessv1.File_browser_automation_studio_v1_readiness_readiness_proto

type Deps struct {
	Reader readiness.ProfileReader
	Logger *logrus.Logger
}

func Module(d Deps) connectx.ServiceMount {
	if d.Reader == nil {
		panic("readiness.Module requires Deps.Reader")
	}
	if d.Logger == nil {
		panic("readiness.Module requires Deps.Logger")
	}
	path, handler := readinessconnect.NewReadinessServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ readinessconnect.ReadinessServiceHandler = (*service)(nil)
