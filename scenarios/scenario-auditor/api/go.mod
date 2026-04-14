module scenario-auditor

go 1.24.0

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require github.com/traefik/yaegi v0.15.1

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251213004720-97cd9d5aeac2 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli/packages/testkit-go => ../../../packages/testkit-go
