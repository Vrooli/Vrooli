module github.com/vrooli/vrooli-cli-go

go 1.25.0

require (
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/vrooli/vrooli/packages/proto => ../proto
