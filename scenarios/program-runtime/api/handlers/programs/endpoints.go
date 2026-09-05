package programs

import (
	"program-runtime/internal/module"

	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "programs_submit", Method: "POST", Path: programsconnect.ProgramServiceSubmitProgramProcedure, Summary: "Execute a governed program.", Category: "programs"},
	{ID: "programs_get", Method: "POST", Path: programsconnect.ProgramServiceGetProgramProcedure, Summary: "Read a submitted program.", Category: "programs"},
	{ID: "programs_wait", Method: "POST", Path: programsconnect.ProgramServiceWaitForProgramProcedure, Summary: "Block until a program reaches a terminal state.", Category: "programs"},
	{ID: "programs_list", Method: "POST", Path: programsconnect.ProgramServiceListProgramsProcedure, Summary: "List submitted programs.", Category: "programs"},
	{ID: "programs_mine", Method: "POST", Path: programsconnect.ProgramServiceMineFailuresProcedure, Summary: "Summarize recurring program failures.", Category: "programs"},
	{ID: "programs_mine_refusals", Method: "POST", Path: programsconnect.ProgramServiceMineRefusalsProcedure, Summary: "Summarize durable binding refusals.", Category: "programs"},
	{ID: "programs_mine_unresolved", Method: "POST", Path: programsconnect.ProgramServiceMineUnresolvedBindingsProcedure, Summary: "Summarize attempted names with no governed binding.", Category: "programs"},
	{ID: "programs_governance_share", Method: "POST", Path: programsconnect.ProgramServiceGovernanceShareProcedure, Summary: "Report governed versus observed program calls over a window.", Category: "programs"},
	{ID: "programs_authoring_eval", Method: "POST", Path: programsconnect.ProgramServiceRunAuthoringEvalProcedure, Summary: "Measure first-attempt authoring against the versioned corpus.", Category: "programs"},
	{ID: "programs_discovery_eval", Method: "POST", Path: programsconnect.ProgramServiceRunDiscoveryEvalProcedure, Summary: "Measure intent discovery against the versioned corpus.", Category: "programs"},
}
