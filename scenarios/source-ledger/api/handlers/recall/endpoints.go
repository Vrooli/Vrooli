package recall

import (
	"source-ledger/internal/module"

	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "recall_query", Path: recallconnect.RecallServiceRecallProcedure, Method: "POST", Summary: "Recall semantic memory", Category: "recall"},
	{ID: "recall_wake", Path: recallconnect.RecallServiceWakeProcedure, Method: "POST", Summary: "Render bounded wake context", Category: "recall"},
	{ID: "recall_zoom", Path: recallconnect.RecallServiceZoomProcedure, Method: "POST", Summary: "Zoom into a memory node", Category: "recall"},
	{ID: "recall_siblings", Path: recallconnect.RecallServiceListSiblingEventsProcedure, Method: "POST", Summary: "List sibling events", Category: "recall"},
}
