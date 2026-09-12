// Package uxmetrics hosts the BAS UXMetricsService Connect-RPC handler.
//
// UXMetricsService exposes computed friction analytics for executions and
// workflow aggregates. All methods require entitlement.TierPro or higher;
// the gating check is performed inside the handler so that callers without
// entitlement context (e.g. CLI) get a structured Connect error rather than
// a 403.
package uxmetrics

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"

	uxmetricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics/uxmetricsconnect"
)

// Deps wires the uxmetrics handler.
type Deps struct {
	// Service is the uxmetrics subsystem facade. Required.
	Service uxmetrics.Service
	// Logger is required.
	Logger *logrus.Logger
}

// Module builds the UXMetricsService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("uxmetrics.Module requires Deps.Logger")
	}
	if d.Service == nil {
		panic("uxmetrics.Module requires Deps.Service")
	}
	path, handler := uxmetricsconnect.NewUXMetricsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ uxmetricsconnect.UXMetricsServiceHandler = (*service)(nil)
