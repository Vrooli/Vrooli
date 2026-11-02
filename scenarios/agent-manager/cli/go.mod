module agent-manager/cli

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	agent-manager v0.0.0
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace agent-manager => ../api

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/searchregister-go => ../../../packages/searchregister-go

replace device-control => ../../device-control/api

replace device-studio => ../../device-studio/api

replace device-studio/cli => ../../device-studio/cli

replace github.com/vrooli/browser-automation-studio => ../../browser-automation-studio/api

replace github.com/vrooli/freshness-go => ../../../packages/freshness-go

replace github.com/vrooli/mdns-go => ../../../packages/mdns-go

replace github.com/vrooli/pyenv-go => ../../../packages/pyenv-go

replace github.com/vrooli/scenarioconfig-go => ../../../packages/scenarioconfig-go

replace github.com/vrooli/ssh-core => ../../../packages/ssh-core

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/vrooli/packages/capability-registry-go => ../../../packages/capability-registry-go

replace github.com/vrooli/vrooli/packages/capabilityprobe => ../../../packages/capabilityprobe

replace github.com/vrooli/vrooli/packages/delivery-ramp-go => ../../../packages/delivery-ramp-go

replace github.com/vrooli/vrooli/scenarios/audio-tools/clients/go => ../../audio-tools/clients/go

replace github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api => ../../scenario-dependency-analyzer/api

replace github.com/vrooli/vrooli/scenarios/system-monitor/api => ../../system-monitor/api

replace github.com/vrooli/vrooli/scenarios/template-manager/api => ../../template-manager/api

replace github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api => ../../vrooli-autoheal/api

replace github.com/vrooli/vrooli/scenarios/vrooli-events => ../../vrooli-events

replace landing-page-business-suite-api => ../../landing-page-business-suite/api

replace notification-hub => ../../notification-hub/api

replace nutrition-planner => ../../nutrition-planner/api

replace personal-planner => ../../personal-planner/api

replace personal-planner/cli => ../../personal-planner/cli

replace program-runtime => ../../program-runtime/api

replace prompt-manager => ../../prompt-manager/api

replace prompt-manager/cli => ../../prompt-manager/cli

replace react-component-library => ../../react-component-library/api

replace react-component-library/cli => ../../react-component-library/cli

replace scenario-dependency-analyzer/cli => ../../scenario-dependency-analyzer/cli

replace scenario-to-cloud => ../../scenario-to-cloud/api

replace scenario-to-desktop-api => ../../scenario-to-desktop/api

replace scenario-to-desktop/cli => ../../scenario-to-desktop/cli

replace swarm-manager => ../../swarm-manager/api

replace swarm-manager/cli => ../../swarm-manager/cli

replace test-genie => ../../test-genie/api

replace test-genie/cli => ../../test-genie/cli

replace ui-health => ../../ui-health/api

replace unit-health => ../../unit-health/api

replace unit-health/cli => ../../unit-health/cli

replace vrooli-autoheal-langrecover => ../../vrooli-autoheal/langrecover

replace web-console => ../../web-console/api

replace workflow-health => ../../workflow-health/api

replace workspace-sandbox => ../../workspace-sandbox/api
