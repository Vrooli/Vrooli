package library

import (
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	"program-runtime/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "library_list", Path: libraryconnect.LibraryServiceListLibraryProcedure, Method: "POST", Summary: "List versioned library programs.", Category: "library"},
	{ID: "library_get", Path: libraryconnect.LibraryServiceGetLibraryProcedure, Method: "POST", Summary: "Read one library program.", Category: "library"},
	{ID: "library_promote", Path: libraryconnect.LibraryServicePromoteLibraryProcedure, Method: "POST", Summary: "Explicitly promote a successful program.", Category: "library"},
	{ID: "library_set_current", Path: libraryconnect.LibraryServiceSetCurrentLibraryProcedure, Method: "POST", Summary: "Set the current version for new sessions.", Category: "library"},
}
