package frozenreplay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	frozenreplayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/frozen_replay"
	frozenreplayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/frozen_replay/frozen_replay_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type clientFactory func() (frozenreplayconnect.FrozenReplayServiceClient, error)

func Register(manifest []byte, apiClient *cliutil.APIClient) (cliapp.SubcommandGroup, error) {
	return register(manifest, func() (frozenreplayconnect.FrozenReplayServiceClient, error) {
		if apiClient == nil || strings.TrimSpace(apiClient.BaseURL()) == "" {
			return nil, errors.New("test-genie API base URL is not configured")
		}
		return frozenreplayconnect.NewFrozenReplayServiceClient(http.DefaultClient, strings.TrimRight(apiClient.BaseURL(), "/")), nil
	})
}

func register(manifest []byte, client clientFactory) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadFromManifestPrimitives(manifest, "frozen-replay", map[string]cliapp.PrimitiveHandler{
		"FrozenReplayService.CreateFrozenReplayReceipt": cliapp.ProtoMutation(createCall(client), func(_ cliapp.OperationContext, response *frozenreplayv1.CreateFrozenReplayReceiptResponse) cliapp.MutationReport {
			receipt := response.GetReceipt()
			return cliapp.MutationReport{
				Result:      []string{fmt.Sprintf("Frozen replay receipt %s: %s", receipt.GetReceiptId(), receipt.GetDisposition())},
				Changes:     []string{"receipt_digest=" + receipt.GetReceiptDigest(), fmt.Sprintf("idempotent_replay=%t", response.GetReplayed())},
				NextCommand: []string{"test-genie frozen-replay get " + receipt.GetReceiptId()},
			}
		}),
		"FrozenReplayService.GetFrozenReplayReceipt": cliapp.ProtoList(getCall(client), func(_ cliapp.OperationContext, response *frozenreplayv1.GetFrozenReplayReceiptResponse) cliapp.ListReport {
			receipt := response.GetReceipt()
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Frozen replay %s: %s", receipt.GetReceiptId(), receipt.GetDisposition())}, Results: []string{"case_id=" + receipt.GetAdjudication().GetCaseId(), "project_id=" + receipt.GetAdjudication().GetProjectId(), "receipt_digest=" + receipt.GetReceiptDigest()}}
		}),
	})
}

func createCall(factory clientFactory) func(cliapp.OperationContext) (*frozenreplayv1.CreateFrozenReplayReceiptResponse, error) {
	return func(operation cliapp.OperationContext) (*frozenreplayv1.CreateFrozenReplayReceiptResponse, error) {
		input := operation.Flag("adjudication")
		if strings.HasPrefix(input, "@") {
			content, err := cliutil.ReadFileString(strings.TrimPrefix(input, "@"))
			if err != nil {
				return nil, fmt.Errorf("read frozen replay adjudication: %w", err)
			}
			input = content
		}
		adjudication := &frozenreplayv1.FrozenReplayAdjudication{}
		if err := protojson.Unmarshal([]byte(input), adjudication); err != nil {
			return nil, fmt.Errorf("decode frozen replay adjudication: %w", err)
		}
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.CreateFrozenReplayReceipt(context.Background(), connect.NewRequest(&frozenreplayv1.CreateFrozenReplayReceiptRequest{IdempotencyKey: operation.Flag("idempotency-key"), Adjudication: adjudication}))
		if err != nil {
			return nil, cliapp.WrapAPIError("create frozen replay receipt", err, nil)
		}
		return response.Msg, nil
	}
}

func getCall(factory clientFactory) func(cliapp.OperationContext) (*frozenreplayv1.GetFrozenReplayReceiptResponse, error) {
	return func(operation cliapp.OperationContext) (*frozenreplayv1.GetFrozenReplayReceiptResponse, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.GetFrozenReplayReceipt(context.Background(), connect.NewRequest(&frozenreplayv1.GetFrozenReplayReceiptRequest{ReceiptId: operation.Positional("receipt_id")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get frozen replay receipt", err, nil)
		}
		return response.Msg, nil
	}
}
