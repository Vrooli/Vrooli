package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/tasks"
)

// Delivery uses the same governed declared runner as direct library calls.
// The archived finish digest and exact checkpoint inputs are the entire retry;
// the consumer's domain program is never submitted here.
func Delivery(runner libraryconnect.LibraryServiceHandler) tasks.Deliver {
	return func(ctx context.Context, record tasks.Record) (map[string]any, error) {
		if record.FinishDigest == "" || record.FinishInputs == nil {
			return nil, fmt.Errorf("missing pinned finish intent")
		}
		provenance := programsv1.Provenance_PROVENANCE_TEST
		if record.Provenance == "operator" {
			provenance = programsv1.Provenance_PROVENANCE_OPERATOR
		} else if record.Provenance == "agent" {
			provenance = programsv1.Provenance_PROVENANCE_AGENT
		} else if record.Provenance != "test" {
			return nil, fmt.Errorf("unsupported memory provenance")
		}
		inputs, err := structpb.NewStruct(record.FinishInputs)
		if err != nil {
			return nil, err
		}
		response, err := runner.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{
			Name: "vrooli-memory.finish-attempt", ExpectedDigest: record.FinishDigest, Inputs: inputs, Provenance: provenance,
		}))
		if err != nil {
			return nil, err
		}
		program := response.Msg.GetProgram()
		if !response.Msg.GetTerminal() || program.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
			return nil, fmt.Errorf("finish execution not acknowledged: %s", program.GetFailureDetail())
		}
		var envelope map[string]any
		if err = json.Unmarshal([]byte(program.GetStdout()), &envelope); err != nil {
			return nil, fmt.Errorf("decode finish envelope: %w", err)
		}
		return envelope, nil
	}
}

// FinishResolver allows optional Memory installation to recover previously
// unpinned captures, while retaining the selected contract before delivery.
func FinishResolver(index *contracts.Index, repository *library.Repository) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		c, ok := index.Get("vrooli-memory", "finish-attempt")
		if !ok || c.ValidationError != "" {
			return "", fmt.Errorf("Memory finish contract unavailable")
		}
		if err := repository.RetainDeclared(ctx, c); err != nil {
			return "", err
		}
		return c.Digest, nil
	}
}
