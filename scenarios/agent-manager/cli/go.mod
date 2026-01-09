module agent-manager/cli

go 1.24.0

toolchain go1.24.11

require (
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251213004720-97cd9d5aeac2 // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
