module github.com/vrooli/browser-automation-studio/bas/seeds

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/vrooli/api-core v0.0.0
)

replace github.com/vrooli/vrooli/packages/proto => ../../../../packages/proto

replace github.com/vrooli/api-core => ../../../../packages/api-core

replace github.com/vrooli/platform-go => ../../../../packages/platform-go

replace github.com/vrooli/repo-contract-go => ../../../../packages/repo-contract-go

replace github.com/vrooli/cli-core => ../../../../packages/cli-core

replace github.com/vrooli/envkit-go => ../../../../packages/envkit-go
