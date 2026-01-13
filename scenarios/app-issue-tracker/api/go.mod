module app-issue-tracker-api

go 1.24.0

toolchain go1.24.9

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/protobuf v1.36.11
	golang.org/x/text v0.30.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/vrooli/api-core => ../../../packages/api-core
replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
