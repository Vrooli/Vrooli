module github.com/vrooli/vrooli/packages/delivery-ramp-go

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.11.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/vrooli/packages/proto => ../proto

replace github.com/vrooli/api-core => ../api-core

replace github.com/vrooli/binaryfetch => ../binaryfetch

replace github.com/vrooli/cli-core => ../cli-core

replace github.com/vrooli/envkit-go => ../envkit-go

replace github.com/vrooli/platform-go => ../platform-go

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenarios/scenario-to-desktop/runtime
