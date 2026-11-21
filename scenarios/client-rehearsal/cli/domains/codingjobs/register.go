package codingjobs

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs/codingjobs_v1connect"
)

const GroupName = "coding-jobs"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	prefix := connect.CodingJobServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		prefix + "CreateCodingJob": cliapp.ProtoMutation(h.createCall, h.createReport),
		prefix + "RecordCodingJobStage": cliapp.ProtoMutation(h.stageCall, h.stageReport),
		prefix + "RecordCodingJobEvaluation": cliapp.ProtoMutation(h.evaluationCall, h.evaluationReport),
		prefix + "GetCodingJob": cliapp.ProtoList(h.getCall, h.getReport),
		prefix + "ListCodingJobs": cliapp.ProtoList(h.listCall, h.listReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil { return cliapp.SubcommandGroup{}, fmt.Errorf("coding jobs: load from manifest: %w", err) }
	return group, nil
}
