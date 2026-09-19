package contextcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture/contextcapture_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, method := range []string{"Import", "Read", "Render", "Delete", "ReconcileImport", "CancelImport"} {
		bindings["ContextCaptureService."+method] = func(run cliapp.RunContext) error {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			client := rpc.NewContextCaptureServiceClient(&http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, core.APIRootBase(), connect.WithReadMaxBytes(45*1024*1024))
			id := run.Flag("id")
			if method == "ReconcileImport" || method == "CancelImport" {
				id = run.Flag("request-id")
			}
			result, err := call(ctx, client, method, run.Flag("request"), id, run.Flag("output"), run.Flag("access-token-file"))
			if err != nil {
				return cliapp.WrapAPIError("context "+strings.ToLower(method), err, nil)
			}
			summary := "Context operation completed."
			if status, ok := result.(*wire.ReconcileImportResponse); ok {
				summary = fmt.Sprintf("Import %s: %s.", status.RequestId, status.State.String())
			}
			if doc, ok := result.(*wire.Document); ok {
				summary = fmt.Sprintf("Context %s (%s).", doc.Id, doc.OriginalSha256)
			}
			return cliapp.RenderProtoMutation(run, result, cliapp.MutationReport{Result: []string{summary}})
		}
	}
	return cliapp.LoadFromManifest(manifest, "context", bindings)
}

func privateFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("context input must be a private regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("context input unavailable")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("context input changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("context input exceeds limit")
	}
	return data, nil
}

func call(ctx context.Context, client rpc.ContextCaptureServiceClient, method, request, id, output, tokenPath string) (proto.Message, error) {
	secret, err := privateFile(tokenPath, 16384)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(string(secret))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("invalid access token file")
	}
	switch method {
	case "CancelImport":
		req := connect.NewRequest(&wire.ReconcileImportRequest{RequestId: id})
		req.Header().Set("Authorization", "Bearer "+token)
		response, err := client.CancelImport(ctx, req)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil

	case "ReconcileImport":
		req := connect.NewRequest(&wire.ReconcileImportRequest{RequestId: id})
		req.Header().Set("Authorization", "Bearer "+token)
		response, err := client.ReconcileImport(ctx, req)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "Import":
		data, err := privateFile(request, 45*1024*1024)
		if err != nil {
			return nil, err
		}
		input := &wire.ImportRequest{}
		if protojson.Unmarshal(data, input) != nil {
			return nil, errors.New("invalid context import JSON")
		}
		req := connect.NewRequest(input)
		req.Header().Set("Authorization", "Bearer "+token)
		response, err := client.Import(ctx, req)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "Read", "Render":
		if id == "" || output == "" {
			return nil, errors.New("context id and new output path required")
		}
		req := connect.NewRequest(&wire.ReferenceRequest{Id: id})
		req.Header().Set("Authorization", "Bearer "+token)
		var doc *wire.Document
		var pixels []byte
		var expectedHash string
		var receipt proto.Message
		if method == "Render" {
			response, err := client.Render(ctx, req)
			if err != nil {
				return nil, err
			}
			doc, pixels, expectedHash = response.Msg.Document, response.Msg.Png, response.Msg.RenderedSha256
			receipt = &wire.RenderResponse{Document: doc, RenderedSha256: expectedHash}
		} else {
			response, err := client.Read(ctx, req)
			if err != nil {
				return nil, err
			}
			doc, pixels = response.Msg.Document, response.Msg.Png
			if doc != nil {
				expectedHash = doc.OriginalSha256
			}
			receipt = doc
		}
		digest := sha256.Sum256(pixels)
		if doc == nil || doc.Id != id || doc.ExpiresAt == nil || doc.ExpiresAt.CheckValid() != nil || !time.Now().Before(doc.ExpiresAt.AsTime()) || len(pixels) == 0 || len(pixels) > 32*1024*1024 || expectedHash != hex.EncodeToString(digest[:]) {
			return nil, errors.New("invalid context image response")
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, errors.New("context output must be a new writable file")
		}
		_, writeErr := file.Write(pixels)
		syncErr := file.Sync()
		closeErr := file.Close()
		if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
			_ = os.Remove(output)
			return nil, errors.New("context image write failed")
		}
		// Pixels are saved privately, never printed as base64 by --json.
		return receipt, nil
	case "Delete":
		if id == "" {
			return nil, errors.New("context id required")
		}
		req := connect.NewRequest(&wire.ReferenceRequest{Id: id})
		req.Header().Set("Authorization", "Bearer "+token)
		response, err := client.Delete(ctx, req)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	default:
		return nil, errors.New("unsupported context operation")
	}
}
