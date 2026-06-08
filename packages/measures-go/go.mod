module github.com/vrooli/measures-go

go 1.25.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1
	github.com/vrooli/aisearch-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/sync v0.20.0 // indirect

replace github.com/vrooli/vrooli/packages/proto => ../proto

replace github.com/vrooli/aisearch-go => ../aisearch-go
