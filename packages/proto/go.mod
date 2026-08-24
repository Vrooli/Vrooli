module github.com/vrooli/vrooli/packages/proto

go 1.25.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1
	connectrpc.com/connect v1.19.2
	github.com/vrooli/platform-go v0.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/sys v0.44.0 // indirect

replace github.com/vrooli/platform-go => ../platform-go
