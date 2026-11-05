module secrets-manager/cli

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mikelolasagasti/xz v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/binaryfetch v0.0.0 // indirect
	github.com/vrooli/cliresolve v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/platform-go v0.0.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.50.1 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/browser-automation-studio => ../../browser-automation-studio/api

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/scenarioconfig-go => ../../../packages/scenarioconfig-go

replace github.com/vrooli/searchregister-go => ../../../packages/searchregister-go

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/vrooli/packages/capability-registry-go => ../../../packages/capability-registry-go

replace nutrition-planner => ../../nutrition-planner/api

replace workflow-health => ../../workflow-health/api
