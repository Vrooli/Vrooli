package programs

import (
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"program-runtime/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "programs_submit", Method: "POST", Path: programsconnect.ProgramServiceSubmitProgramProcedure, Summary: "Execute a governed program.", Category: "programs"},
	{ID: "programs_get", Method: "POST", Path: programsconnect.ProgramServiceGetProgramProcedure, Summary: "Read a submitted program.", Category: "programs"},
	{ID: "programs_list", Method: "POST", Path: programsconnect.ProgramServiceListProgramsProcedure, Summary: "List submitted programs.", Category: "programs"},
	{ID: "programs_mine", Method: "POST", Path: programsconnect.ProgramServiceMineFailuresProcedure, Summary: "Summarize recurring program failures.", Category: "programs"},
	{ID: "programs_mine_refusals", Method: "POST", Path: programsconnect.ProgramServiceMineRefusalsProcedure, Summary: "Summarize durable binding refusals.", Category: "programs"},
	{ID: "programs_mine_unresolved", Method: "POST", Path: programsconnect.ProgramServiceMineUnresolvedBindingsProcedure, Summary: "Summarize attempted names with no governed binding.", Category: "programs"},
}
