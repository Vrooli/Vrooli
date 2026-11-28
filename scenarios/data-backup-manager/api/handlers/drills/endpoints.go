package drills

import (
	"data-backup-manager/internal/modulecontract"

	drillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills/drills_v1connect"
)

var Endpoints = []modulecontract.EndpointDescriptor{
	{ID: "drills_preview", Path: drillsconnect.RecoveryDrillsServicePreviewDrillProcedure, Method: "POST", Summary: "Preview a recovery drill", Description: "Read-only eligibility and latest-snapshot preview; does not restore or mutate data.", Category: "drills"},
	{ID: "drills_run", Path: drillsconnect.RecoveryDrillsServiceRunDrillProcedure, Method: "POST", Summary: "Run a recovery drill", Description: "Durably records and asynchronously verifies the latest successful snapshot in scratch.", Category: "drills"},
	{ID: "drills_get", Path: drillsconnect.RecoveryDrillsServiceGetDrillProcedure, Method: "POST", Summary: "Get a recovery drill", Description: "Returns persisted drill status and linked restore evidence.", Category: "drills"},
	{ID: "drills_list", Path: drillsconnect.RecoveryDrillsServiceListDrillsProcedure, Method: "POST", Summary: "List recovery drills", Description: "Lists persisted recovery-drill evidence newest first with an opaque continuation token for complete history.", Category: "drills"},
	{ID: "drills_retire", Path: drillsconnect.RecoveryDrillsServiceRetireDrillProcedure, Method: "POST", Summary: "Retire a completed recovery drill", Description: "Marks a completed drill as retired while preserving its original status and durable evidence.", Category: "drills"},
}
