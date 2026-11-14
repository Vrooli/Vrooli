module github.com/vrooli/vrooli

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/klauspost/compress v1.18.7
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/stretchr/testify v1.12.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/binaryfetch v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/cliresolve v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/platform-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime v0.0.0
	golang.org/x/mod v0.41.0
	golang.org/x/sys v0.48.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	github.com/andybalholm/brotli v1.2.0
	golang.org/x/sync v0.23.0 // indirect
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mikelolasagasti/xz v1.0.1
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/vrooli/ai-go v0.0.0 // indirect
	github.com/vrooli/browser-automation-studio v0.0.0
	github.com/vrooli/maturity-go v0.0.0
	github.com/vrooli/measures-go v0.0.0
	github.com/vrooli/vrooli/packages/capability-registry-go v0.0.0
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	nutrition-planner v0.0.0
	workflow-health v0.0.0
)

replace github.com/vrooli/api-core => ./packages/api-core

replace github.com/vrooli/cliresolve => ./packages/cliresolve

replace github.com/vrooli/binaryfetch => ./packages/binaryfetch

replace github.com/vrooli/cli-core => ./packages/cli-core

replace github.com/vrooli/platform-go => ./packages/platform-go

replace github.com/vrooli/repo-contract-go => ./packages/repo-contract-go

replace github.com/vrooli/envkit-go => ./packages/envkit-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ./scenarios/scenario-to-desktop/runtime

replace github.com/vrooli/vrooli/packages/proto => ./packages/proto

replace agent-manager => ./scenarios/agent-manager/api

replace agent-manager/cli => ./scenarios/agent-manager/cli

replace device-control => ./scenarios/device-control/api

replace device-studio => ./scenarios/device-studio/api

replace device-studio/cli => ./scenarios/device-studio/cli

replace github.com/vrooli/ai-go => ./packages/ai-go

replace github.com/vrooli/browser-automation-studio => ./scenarios/browser-automation-studio/api

replace github.com/vrooli/freshness-go => ./packages/freshness-go

replace github.com/vrooli/maturity-go => ./packages/maturity-go

replace github.com/vrooli/measures-go => ./packages/measures-go

replace github.com/vrooli/pyenv-go => ./packages/pyenv-go

replace github.com/vrooli/searchregister-go => ./packages/searchregister-go

replace github.com/vrooli/vrooli/packages/capability-registry-go => ./packages/capability-registry-go

replace github.com/vrooli/vrooli/packages/delivery-ramp-go => ./packages/delivery-ramp-go

replace github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api => ./scenarios/scenario-dependency-analyzer/api

replace github.com/vrooli/vrooli/scenarios/system-monitor/api => ./scenarios/system-monitor/api

replace github.com/vrooli/vrooli/scenarios/template-manager/api => ./scenarios/template-manager/api

replace github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api => ./scenarios/vrooli-autoheal/api

replace github.com/vrooli/vrooli/scenarios/vrooli-events => ./scenarios/vrooli-events

replace landing-page-business-suite-api => ./scenarios/landing-page-business-suite/api

replace notification-hub => ./scenarios/notification-hub/api

replace nutrition-planner => ./scenarios/nutrition-planner/api

replace personal-planner => ./scenarios/personal-planner/api

replace personal-planner/cli => ./scenarios/personal-planner/cli

replace program-runtime => ./scenarios/program-runtime/api

replace prompt-manager => ./scenarios/prompt-manager/api

replace prompt-manager/cli => ./scenarios/prompt-manager/cli

replace react-component-library => ./scenarios/react-component-library/api

replace react-component-library/cli => ./scenarios/react-component-library/cli

replace scenario-dependency-analyzer/cli => ./scenarios/scenario-dependency-analyzer/cli

replace scenario-to-cloud => ./scenarios/scenario-to-cloud/api

replace scenario-to-desktop-api => ./scenarios/scenario-to-desktop/api

replace scenario-to-desktop/cli => ./scenarios/scenario-to-desktop/cli

replace swarm-manager => ./scenarios/swarm-manager/api

replace swarm-manager/cli => ./scenarios/swarm-manager/cli

replace test-genie => ./scenarios/test-genie/api

replace test-genie/cli => ./scenarios/test-genie/cli

replace ui-health => ./scenarios/ui-health/api

replace unit-health => ./scenarios/unit-health/api

replace unit-health/cli => ./scenarios/unit-health/cli

replace web-console => ./scenarios/web-console/api

replace workflow-health => ./scenarios/workflow-health/api

replace workspace-sandbox => ./scenarios/workspace-sandbox/api

replace github.com/vrooli/mdns-go => ./packages/mdns-go

replace github.com/vrooli/scenarioconfig-go => ./packages/scenarioconfig-go

replace github.com/vrooli/ssh-core => ./packages/ssh-core

replace github.com/vrooli/vrooli-cli-go => ./packages/vrooli-cli-go

replace github.com/vrooli/vrooli/packages/capabilityprobe => ./packages/capabilityprobe

replace github.com/vrooli/vrooli/scenarios/audio-tools/clients/go => ./scenarios/audio-tools/clients/go

replace vrooli-autoheal-langrecover => ./scenarios/vrooli-autoheal/langrecover
