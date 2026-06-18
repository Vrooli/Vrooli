module github.com/vrooli/cli-core

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/vrooli/packages/proto v0.0.0
)

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..

replace github.com/vrooli/vrooli/packages/proto => ../proto
