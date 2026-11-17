package vectorgenerations

import (
	vectorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/vector/vector_v1connect"
	"source-ledger/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "vector_generation_stage", Path: vectorconnect.VectorGenerationServiceStageGenerationProcedure, Method: "POST", Summary: "Stage a recipe-bound SQLite vector generation", Category: "ops"},
	{ID: "vector_generation_catch_up", Path: vectorconnect.VectorGenerationServiceCatchUpGenerationProcedure, Method: "POST", Summary: "Catch up a staged SQLite vector generation", Category: "ops"},
	{ID: "vector_generation_validate", Path: vectorconnect.VectorGenerationServiceValidateGenerationProcedure, Method: "POST", Summary: "Validate source coverage and vector integrity before cutover", Category: "ops"},
	{ID: "vector_generation_promote", Path: vectorconnect.VectorGenerationServicePromoteGenerationProcedure, Method: "POST", Summary: "Promote a validated SQLite vector generation", Category: "ops"},
	{ID: "vector_generation_rollback", Path: vectorconnect.VectorGenerationServiceRollbackGenerationProcedure, Method: "POST", Summary: "Restore the retained SQLite vector generation", Category: "ops"},
	{ID: "vector_generation_operation", Path: vectorconnect.VectorGenerationServiceGetGenerationOperationProcedure, Method: "POST", Summary: "Read durable SQLite generation operation status", Category: "ops"},
	{ID: "vector_generation_resume", Path: vectorconnect.VectorGenerationServiceResumeGenerationOperationProcedure, Method: "POST", Summary: "Resume an interrupted SQLite generation operation", Category: "ops"},
	{ID: "vector_generation_active", Path: vectorconnect.VectorGenerationServiceGetActiveGenerationProcedure, Method: "POST", Summary: "Read the active SQLite vector generation", Category: "ops"},
}
