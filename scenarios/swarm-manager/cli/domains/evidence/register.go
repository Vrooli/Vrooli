package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

// Register exposes the canonical evidence ledger through verified
// renderer-separated Connect primitives.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "evidence", Description: "Query and reconcile canonical run evidence", Subcommands: []cliapp.Command{listRunCommand(), listEntityCommand(), reconcileCommand(), verifyCommand()}}
}

func client(op cliapp.OperationContext) apiconnect.EvidenceServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewEvidenceServiceClient(h, base)
}

func listRunCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "run", NeedsAPI: true, Description: "List evidence for a verified run (--run-id ID) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "run-id", Description: "Verified Agent Manager run ID", Required: true}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.EvidenceListResponse, error) {
		response, err := client(op).ListRun(context.Background(), connect.NewRequest(&apipb.EvidenceListRunRequest{RunId: strings.TrimSpace(op.Flag("run-id"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, evidenceListReport))
}

func listEntityCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "entity", NeedsAPI: true, Description: "List evidence for an entity (--kind KIND --id ID) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Description: "Evidence subject kind", Required: true}, {Name: "id", Description: "Evidence subject ID", Required: true}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.EvidenceListResponse, error) {
		response, err := client(op).ListEntity(context.Background(), connect.NewRequest(&apipb.EvidenceListEntityRequest{SubjectKind: strings.TrimSpace(op.Flag("kind")), SubjectId: strings.TrimSpace(op.Flag("id"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, evidenceListReport))
}

func evidenceListReport(_ cliapp.OperationContext, response *apipb.EvidenceListResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.GetRecords()))
	for _, record := range response.GetRecords() {
		rows = append(rows, fmt.Sprintf("%s %s/%s %s [%s, %s]", record.GetRunId(), record.GetSubjectKind(), record.GetSubjectId(), record.GetAction(), record.GetConfidence(), record.GetVerification()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Evidence records: %d", len(rows))}, ResultsHeading: "Evidence", Results: rows}
}

func reconcileCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "reconcile", NeedsAPI: true, Description: "Retry evidence producers for a run (--run-id ID) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "run-id", Description: "Verified Agent Manager run ID", Required: true}}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.EvidenceReconcileResponse, error) {
		response, err := client(op).Reconcile(context.Background(), connect.NewRequest(&apipb.EvidenceReconcileRequest{RunId: strings.TrimSpace(op.Flag("run-id"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *apipb.EvidenceReconcileResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Evidence for %s: %s", response.GetRunId(), response.GetStatus())}}
	}))
}

func verifyCommand() cliapp.Command {
	flags := []cliapp.Flag{{Name: "owner-kind", Description: "agent_session or operating_mode_execution", Required: true}, {Name: "owner-id", Description: "Owner ID", Required: true}, {Name: "round", Description: "Operating-mode round", Default: "0"}, {Name: "event-id", Description: "Stable operator event ID", Required: true}, {Name: "run-id", Description: "Run ID", Required: true}, {Name: "subject-kind", Description: "Evidence subject kind", Required: true}, {Name: "subject-id", Description: "Evidence subject ID", Required: true}, {Name: "action", Description: "Observed action", Required: true}, {Name: "actor", Description: "Operator identity", Required: true}, {Name: "reason", Description: "Verification reason", Required: true}, {Name: "metadata", Description: "Optional JSON object of bounded metadata"}}
	cmd := cliapp.Command{Name: "verify", NeedsAPI: true, Description: "Append an operator-verified observation (--owner-kind K --owner-id ID --event-id ID --run-id ID --subject-kind K --subject-id ID --action A --actor ID --reason TEXT) [--metadata JSON] [--json]", Args: cliapp.ArgSchema{Flags: flags}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.EvidenceRecord, error) {
		round, err := strconv.ParseInt(op.Flag("round"), 10, 32)
		if err != nil || round < 0 {
			return nil, fmt.Errorf("--round must be a non-negative integer")
		}
		metadata, err := parseMetadata(op.Flag("metadata"))
		if err != nil {
			return nil, err
		}
		response, err := client(op).RecordOperatorVerification(context.Background(), connect.NewRequest(&apipb.EvidenceOperatorVerificationRequest{OwnerKind: strings.TrimSpace(op.Flag("owner-kind")), OwnerId: strings.TrimSpace(op.Flag("owner-id")), OwnerRound: int32(round), EventId: strings.TrimSpace(op.Flag("event-id")), RunId: strings.TrimSpace(op.Flag("run-id")), SubjectKind: strings.TrimSpace(op.Flag("subject-kind")), SubjectId: strings.TrimSpace(op.Flag("subject-id")), Action: strings.TrimSpace(op.Flag("action")), Actor: strings.TrimSpace(op.Flag("actor")), Reason: strings.TrimSpace(op.Flag("reason")), Metadata: metadata}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, _ *apipb.EvidenceRecord) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Operator verification recorded."}}
	}))
}

func parseMetadata(raw string) (*structpb.Struct, error) {
	values := map[string]string{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("--metadata must be a JSON object with string values: %w", err)
		}
	}
	fields := make(map[string]any, len(values))
	for key, value := range values {
		fields[key] = value
	}
	return structpb.NewStruct(fields)
}
