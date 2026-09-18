package inventory

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/inventory/inventory_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "inventory_list_events", Path: connect.InventoryServiceListEventsProcedure, Method: "POST", Summary: "List inventory events", Category: "inventory"},
	{ID: "inventory_record_event", Path: connect.InventoryServiceRecordEventProcedure, Method: "POST", Summary: "Record an inventory event", Category: "inventory"},
	{ID: "inventory_prepare_batch", Path: connect.InventoryServicePrepareBatchProcedure, Method: "POST", Summary: "Prepare a batch and consume raw stock once", Category: "inventory"},
	{ID: "inventory_consume_batch_portion", Path: connect.InventoryServiceConsumeBatchPortionProcedure, Method: "POST", Summary: "Consume one prepared batch portion", Category: "inventory"},
	{ID: "inventory_undo_batch_portion", Path: connect.InventoryServiceUndoBatchPortionProcedure, Method: "POST", Summary: "Undo one prepared batch portion", Category: "inventory"},
	{ID: "inventory_stage_receipt_proposal", Path: connect.InventoryServiceStageReceiptProposalProcedure, Method: "POST", Summary: "Stage a receipt purchase proposal for review", Category: "inventory"},
	{ID: "inventory_list_receipt_proposals", Path: connect.InventoryServiceListReceiptProposalsProcedure, Method: "POST", Summary: "List staged receipt purchase proposals", Category: "inventory"},
	{ID: "inventory_apply_receipt_proposal", Path: connect.InventoryServiceApplyReceiptProposalProcedure, Method: "POST", Summary: "Apply an approved receipt purchase proposal", Category: "inventory"},
}
