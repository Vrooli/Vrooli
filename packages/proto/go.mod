module github.com/vrooli/vrooli/packages/proto

go 1.25.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1
	connectrpc.com/connect v1.19.2
	github.com/vrooli/platform-go v0.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
)

replace github.com/vrooli/platform-go => ../platform-go

// platform-go uses the repository's local contract module. Keep this
// replacement explicit because Go does not inherit replace directives from a
// dependency module when packages/proto is built independently.
replace github.com/vrooli/repo-contract-go => ../repo-contract-go
