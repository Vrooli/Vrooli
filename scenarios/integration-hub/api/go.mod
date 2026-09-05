module github.com/vrooli/vrooli/scenarios/integration-hub/api

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
