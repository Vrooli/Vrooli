module github.com/vrooli/browser-automation-studio/bas/seeds

go 1.24.0

require (
	connectrpc.com/connect v1.19.2
	github.com/vrooli/vrooli/packages/proto v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect

replace github.com/vrooli/vrooli/packages/proto => ../../../../packages/proto
