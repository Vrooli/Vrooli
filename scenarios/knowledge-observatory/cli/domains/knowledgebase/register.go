package knowledgebase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	koconnect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func invoke[Q any, R any](call func(context.Context, *connect.Request[Q]) (*connect.Response[R], error), newRequest func() *Q, fields []string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		values := map[string]any{}
		for _, field := range fields {
			if field == "allow-missing" || field == "skip-external-links" {
				values[strings.ReplaceAll(field, "-", "_")] = ctx.BoolFlag(field)
				continue
			}
			value := ctx.Flag(field)
			if value == "" {
				continue
			}
			key := strings.ReplaceAll(field, "-", "_")
			switch field {
			case "limit", "offset", "max-files":
				n, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid %s: %w", field, err)
				}
				values[key] = n
			case "paths", "checks":
				values[key] = strings.Split(value, ",")
			default:
				values[key] = value
			}
		}
		msg := newRequest()
		wire, _ := json.Marshal(values)
		if err := protojson.Unmarshal(wire, any(msg).(proto.Message)); err != nil {
			return err
		}
		response, err := call(context.Background(), connect.NewRequest(msg))
		if err != nil {
			return err
		}
		payload := any(response.Msg).(proto.Message)
		body, err := protojson.MarshalOptions{Multiline: true, UseProtoNames: true}.Marshal(payload)
		if err != nil {
			return err
		}
		return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: []string{"Read-only knowledge evidence; source authority requires review."}, Results: []string{string(body)}})
	}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	client := koconnect.NewKnowledgeBaseServiceClient(httpClient, base)
	health := koconnect.NewKnowledgeObservatoryServiceClient(httpClient, base)
	return cliapp.LoadFromManifest(manifest, "knowledge-base", map[string]func(cliapp.RunContext) error{
		"KnowledgeBaseService.SearchDocuments":  invoke(client.SearchDocuments, func() *kov1.SearchDocumentsRequest { return &kov1.SearchDocumentsRequest{} }, []string{"query", "scope", "target", "mode", "limit"}),
		"KnowledgeBaseService.InspectDocument":  invoke(client.InspectDocument, func() *kov1.InspectDocumentRequest { return &kov1.InspectDocumentRequest{} }, []string{"path", "offset", "limit", "expected-sha256", "allow-missing"}),
		"KnowledgeBaseService.ReviewDocuments":  invoke(client.ReviewDocuments, func() *kov1.ReviewDocumentsRequest { return &kov1.ReviewDocumentsRequest{} }, []string{"paths", "base-path", "max-files"}),
		"KnowledgeBaseService.KnowledgeStatus":  invoke(client.KnowledgeStatus, func() *kov1.KnowledgeStatusRequest { return &kov1.KnowledgeStatusRequest{} }, nil),
		"KnowledgeObservatoryService.DocHealth": invoke(health.DocHealth, func() *kov1.DocHealthRequest { return &kov1.DocHealthRequest{} }, []string{"scope", "path", "scenario-name", "checks", "skip-external-links"}),
	})
}
