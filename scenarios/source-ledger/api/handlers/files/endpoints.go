package files

import (
	"source-ledger/internal/module"

	filesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files/filesv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "files_search", Path: filesconnect.FilesServiceSearchProcedure, Method: "POST", Summary: "Search a file corpus (research home) with sync-on-read and citations", Category: "files"},
	{ID: "files_feeds", Path: filesconnect.FilesServiceListFeedsProcedure, Method: "POST", Summary: "Summarise a file corpus's feeds: size, freshness and newest files", Category: "files"},
}
