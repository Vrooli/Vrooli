module github.com/vrooli/nodeclient

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/api-core => ../api-core

replace github.com/vrooli/cli-core => ../cli-core

replace github.com/vrooli/envkit-go => ../envkit-go

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli/packages/proto => ../proto
