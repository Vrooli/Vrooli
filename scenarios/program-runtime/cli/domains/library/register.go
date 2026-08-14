package library

import (
	"context"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
)

const GroupName = "library"

type handlers struct {
	client libraryconnect.LibraryServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LibraryService.ListLibrary":       cliapp.ProtoList(h.list, h.listReport),
		"LibraryService.GetLibrary":        cliapp.ProtoList(h.get, h.getReport),
		"LibraryService.PromoteLibrary":    cliapp.ProtoMutation(h.promote, h.promoteReport),
		"LibraryService.SetCurrentLibrary": cliapp.ProtoMutation(h.setCurrent, h.currentReport),
	})
}

func (h *handlers) list(_ cliapp.OperationContext) (*libraryv1.ListLibraryResponse, error) {
	r, err := h.client.ListLibrary(context.Background(), connect.NewRequest(&libraryv1.ListLibraryRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list library", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) get(ctx cliapp.OperationContext) (*libraryv1.GetLibraryResponse, error) {
	version := int64(0)
	if value := ctx.Flag("version"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, err
		}
		version = parsed
	}
	r, err := h.client.GetLibrary(context.Background(), connect.NewRequest(&libraryv1.GetLibraryRequest{Name: ctx.Positional("name"), Version: version}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get library", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) promote(ctx cliapp.OperationContext) (*libraryv1.PromoteLibraryResponse, error) {
	r, err := h.client.PromoteLibrary(context.Background(), connect.NewRequest(&libraryv1.PromoteLibraryRequest{ProgramId: ctx.Flag("program-id"), Name: ctx.Flag("name"), Description: ctx.Flag("description"), PromotedBy: ctx.Flag("promoted-by"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("promote library", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) setCurrent(ctx cliapp.OperationContext) (*libraryv1.SetCurrentLibraryResponse, error) {
	version, err := strconv.ParseInt(ctx.Flag("version"), 10, 64)
	if err != nil || version <= 0 {
		return nil, err
	}
	r, err := h.client.SetCurrentLibrary(context.Background(), connect.NewRequest(&libraryv1.SetCurrentLibraryRequest{Name: ctx.Positional("name"), Version: version}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set current library", err, nil)
	}
	return r.Msg, nil
}
func (*handlers) listReport(_ cliapp.OperationContext, r *libraryv1.ListLibraryResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Library programs: " + strconv.Itoa(len(r.GetPrograms()))}}
}
func (*handlers) getReport(_ cliapp.OperationContext, _ *libraryv1.GetLibraryResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Library program read."}}
}
func (*handlers) promoteReport(_ cliapp.OperationContext, _ *libraryv1.PromoteLibraryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Library program promoted."}}
}
func (*handlers) currentReport(_ cliapp.OperationContext, _ *libraryv1.SetCurrentLibraryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Library version selected for new sessions."}}
}
