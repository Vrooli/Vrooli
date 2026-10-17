package intake

import (
	"document-manager/internal/module"

	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/intake/intake_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "intake_ingest", Path: intakeconnect.IntakeServiceIngestProcedure, Method: "POST", Summary: "Ingest bytes by content hash", Category: "intake"},
	{ID: "intake_get_document", Path: intakeconnect.IntakeServiceGetDocumentProcedure, Method: "POST", Summary: "Get an ingested document", Category: "intake"},
	{ID: "intake_list_documents", Path: intakeconnect.IntakeServiceListDocumentsProcedure, Method: "POST", Summary: "List ingested documents", Category: "intake"},
	{ID: "intake_list_sources", Path: intakeconnect.IntakeServiceListSourcesProcedure, Method: "POST", Summary: "List document sources", Category: "intake"},
	{ID: "intake_configure_watch", Path: intakeconnect.IntakeServiceConfigureWatchProcedure, Method: "POST", Summary: "Configure a watched intake path", Category: "intake"},
	{ID: "intake_get_type_verdict", Path: intakeconnect.IntakeServiceGetTypeVerdictProcedure, Method: "POST", Summary: "Read the stored type verdict", Category: "intake"},
}
