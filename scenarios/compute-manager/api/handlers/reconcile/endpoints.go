package reconcile

import (
	"compute-manager/internal/module"
	reconcileconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile/reconcile_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "reconcile_run", Path: reconcileconnect.ReconcileServiceRunReconciliationProcedure, Method: "POST", Summary: "Run reconciliation", Category: "reconcile"},
	{ID: "reconcile_list", Path: reconcileconnect.ReconcileServiceListFindingsProcedure, Method: "POST", Summary: "List reconciliation findings", Category: "reconcile"},
	{ID: "reconcile_get", Path: reconcileconnect.ReconcileServiceGetFindingProcedure, Method: "POST", Summary: "Get a reconciliation finding", Category: "reconcile"},
	{ID: "reconcile_quarantine", Path: reconcileconnect.ReconcileServiceQuarantineFindingProcedure, Method: "POST", Summary: "Quarantine a reconciliation finding", Category: "reconcile"},
}
