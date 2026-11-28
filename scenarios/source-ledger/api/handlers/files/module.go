package files

import (
	"source-ledger/internal/files"
	"source-ledger/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files"
	filesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files/filesv1connect"
)

// Module mounts the file-corpus search service.
func Module(service *files.Service) module.Module {
	path, handler := filesconnect.NewFilesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "files", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = filesv1.File_source_ledger_v1_files_files_proto
