module swarm-manager

go 1.24.0

toolchain go1.24.12

require (
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
