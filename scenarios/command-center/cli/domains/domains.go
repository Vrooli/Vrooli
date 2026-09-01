package domains

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	read := func(path string) func([]string) error { return func(args []string) error { body, err := core.Get(path, nil); if err != nil { return err }; if hasJSON(args) { _,err=os.Stdout.Write(body); if err==nil {_,err=os.Stdout.Write([]byte("\n"))}; return err }; fmt.Println(string(body)); return nil } }
	room := func(args []string) error { if len(args)==0 || strings.HasPrefix(args[0],"-") { return fmt.Errorf("room requires an id") }; path := "/rooms/"+args[0]; if len(args)>1 && args[1]=="--samples" && len(args)>2 { path += "?samples="+args[2] }; return read(path)(args) }
	return []cliapp.CommandGroup{{Title:"Instrument reads",Commands:[]cliapp.Command{{Name:"board",Description:"Show board shape",NeedsAPI:true,Run:read("/board")},{Name:"room",Description:"Show one composed room",NeedsAPI:true,Run:room},{Name:"focus",Description:"Show ranked findings",NeedsAPI:true,Run:read("/focus")},{Name:"open-loop",Description:"Show dated open holes",NeedsAPI:true,Run:read("/open-loop")},{Name:"describe",Description:"Describe the sensor space",NeedsAPI:true,Run:read("/capabilities/describe")},{Name:"gaps",Description:"Show compatibility gaps",NeedsAPI:true,Run:read("/gaps")}}}}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Prefer domain packages as the default growth path:
//
//	cli/domains/tasks/register.go
//	cli/domains/projects/register.go
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
//   - use cliapp.PrintReportJSON(...) when a --json mode should mirror the
//     same structured report
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}

func hasJSON(args []string) bool { for _,a:=range args {if a=="--json" {return true}}; return false }
