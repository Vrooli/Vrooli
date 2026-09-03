module react-component-library/cli

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/stretchr/testify v1.12.1
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	react-component-library v0.0.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/vrooli/vrooli/packages/capability-registry-go v0.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.50.1 // indirect
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	github.com/vrooli/vrooli v0.0.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace react-component-library => ../api

replace github.com/vrooli/vrooli/packages/capability-registry-go => ../../../packages/capability-registry-go
