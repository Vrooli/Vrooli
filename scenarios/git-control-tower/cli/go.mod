module git-control-tower/cli

go 1.25.0

require github.com/vrooli/cli-core v0.0.0

require (
	connectrpc.com/connect v1.19.2 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace git-control-tower => ../api

replace github.com/vrooli/vrooli => ../../..
