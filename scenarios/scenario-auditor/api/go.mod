module scenario-auditor

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/maturity-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli-cli-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require github.com/traefik/yaegi v0.15.1

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	connectrpc.com/connect v1.19.2 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go
