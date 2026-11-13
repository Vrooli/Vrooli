package rangehandler

import (
	"extraction-range/internal/module"
	rangeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range/range_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "range_list_levels", Path: rangeconnect.RangeServiceListLevelsProcedure, Method: "POST", Summary: "List practice levels", Description: "Lists the data-defined extraction challenges, optionally filtered by family.", Category: "range", Response: &module.Schema{Type: "object", Properties: map[string]string{"levels": "array<LevelSummary>"}}},
	{ID: "range_get_level", Path: rangeconnect.RangeServiceGetLevelProcedure, Method: "POST", Summary: "Get a practice level", Description: "Returns the task brief and owned practice-page URL without revealing its answer key.", Category: "range", Response: &module.Schema{Type: "object", Properties: map[string]string{"level": "LevelSummary", "target_url": "string"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Level does not exist"}}},
	{ID: "range_submit", Path: rangeconnect.RangeServiceSubmitProcedure, Method: "POST", Summary: "Score an extraction submission", Description: "Scores exact record matches with precision, recall and F1, attributed to the submitting tool.", Category: "range", Request: &module.Schema{Type: "object", Properties: map[string]string{"tool": "string", "level_id": "string", "records": "array<ExtractedRecord>"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"f1": "number", "passed": "boolean", "correct": "integer"}}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Tool or level is missing or invalid"}}},
	{ID: "range_scoreboard", Path: rangeconnect.RangeServiceScoreboardProcedure, Method: "POST", Summary: "Read the practice scoreboard", Description: "Returns best scores by tool and level, optionally filtered by tool or family.", Category: "range", Response: &module.Schema{Type: "object", Properties: map[string]string{"scores": "array<SubmitResponse>"}}},
}
