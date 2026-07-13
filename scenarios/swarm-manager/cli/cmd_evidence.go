package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

func (a *App) evidenceClient() apiconnect.EvidenceServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewEvidenceServiceClient(httpClient, baseURL)
}

func (a *App) cmdEvidenceRun(args []string) error {
	fs := flag.NewFlagSet("evidence run", flag.ContinueOnError)
	runID := fs.String("run-id", "", "Verified Agent Manager run ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("run-id", *runID); err != nil {
		return err
	}
	response, err := a.evidenceClient().ListRun(context.Background(), connect.NewRequest(&apipb.EvidenceListRunRequest{RunId: strings.TrimSpace(*runID)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, response.Msg)
	}
	return printEvidence(response.Msg.GetRecords())
}

func (a *App) cmdEvidenceEntity(args []string) error {
	fs := flag.NewFlagSet("evidence entity", flag.ContinueOnError)
	kind := fs.String("kind", "", "Evidence subject kind")
	id := fs.String("id", "", "Evidence subject ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kind, "id", *id); err != nil {
		return err
	}
	response, err := a.evidenceClient().ListEntity(context.Background(), connect.NewRequest(&apipb.EvidenceListEntityRequest{SubjectKind: strings.TrimSpace(*kind), SubjectId: strings.TrimSpace(*id)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, response.Msg)
	}
	return printEvidence(response.Msg.GetRecords())
}

func (a *App) cmdEvidenceReconcile(args []string) error {
	fs := flag.NewFlagSet("evidence reconcile", flag.ContinueOnError)
	runID := fs.String("run-id", "", "Verified Agent Manager run ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("run-id", *runID); err != nil {
		return err
	}
	response, err := a.evidenceClient().Reconcile(context.Background(), connect.NewRequest(&apipb.EvidenceReconcileRequest{RunId: strings.TrimSpace(*runID)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, response.Msg)
	}
	fmt.Printf("Evidence for %s: %s\n", response.Msg.GetRunId(), response.Msg.GetStatus())
	return nil
}

func (a *App) cmdEvidenceVerify(args []string) error {
	fs := flag.NewFlagSet("evidence verify", flag.ContinueOnError)
	ownerKind := fs.String("owner-kind", "", "agent_session or operating_mode_execution")
	ownerID := fs.String("owner-id", "", "Owner ID")
	round := fs.Int("round", 0, "Operating-mode round")
	eventID := fs.String("event-id", "", "Stable operator event ID")
	runID := fs.String("run-id", "", "Run ID")
	subjectKind := fs.String("subject-kind", "", "Evidence subject kind")
	subjectID := fs.String("subject-id", "", "Evidence subject ID")
	action := fs.String("action", "", "Observed action")
	actor := fs.String("actor", "", "Operator identity")
	reason := fs.String("reason", "", "Verification reason")
	metadataJSON := fs.String("metadata", "", "Optional JSON object of bounded metadata")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("owner-kind", *ownerKind, "owner-id", *ownerID, "event-id", *eventID, "run-id", *runID, "subject-kind", *subjectKind, "subject-id", *subjectID, "action", *action, "actor", *actor, "reason", *reason); err != nil {
		return err
	}
	metadata := map[string]string{}
	if strings.TrimSpace(*metadataJSON) != "" {
		if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err != nil {
			return fmt.Errorf("--metadata must be a JSON object with string values: %w", err)
		}
	}
	protoMetadata, err := structpb.NewStruct(stringMapAny(metadata))
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	response, err := a.evidenceClient().RecordOperatorVerification(context.Background(), connect.NewRequest(&apipb.EvidenceOperatorVerificationRequest{OwnerKind: strings.TrimSpace(*ownerKind), OwnerId: strings.TrimSpace(*ownerID), OwnerRound: int32(*round), EventId: strings.TrimSpace(*eventID), RunId: strings.TrimSpace(*runID), SubjectKind: strings.TrimSpace(*subjectKind), SubjectId: strings.TrimSpace(*subjectID), Action: strings.TrimSpace(*action), Actor: strings.TrimSpace(*actor), Reason: strings.TrimSpace(*reason), Metadata: protoMetadata}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, response.Msg)
	}
	fmt.Println("Operator verification recorded.")
	return nil
}

func printEvidence(records []*apipb.EvidenceRecord) error {
	if len(records) == 0 {
		fmt.Println("No evidence found.")
		return nil
	}
	for _, record := range records {
		fmt.Printf("%s %s/%s %s [%s, %s]\n", record.GetRunId(), record.GetSubjectKind(), record.GetSubjectId(), record.GetAction(), record.GetConfidence(), record.GetVerification())
	}
	return nil
}

func stringMapAny(input map[string]string) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
