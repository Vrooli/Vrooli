module github.com/vrooli/agentharness

go 1.25.0

require github.com/vrooli/cli-core v0.0.0

require (
	connectrpc.com/connect v1.19.2 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/vrooli/cli-core => ../cli-core

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..

replace github.com/vrooli/vrooli/packages/proto => ../proto

replace github.com/vrooli/envkit-go => ../envkit-go

replace github.com/vrooli/api-core => ../api-core

replace github.com/vrooli/binaryfetch => ../binaryfetch

replace github.com/vrooli/platform-go => ../platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenarios/scenario-to-desktop/runtime
