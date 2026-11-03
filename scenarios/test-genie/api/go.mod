module test-genie

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/testcontainers/testcontainers-go v0.31.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/freshness-go v0.0.0
	github.com/vrooli/maturity-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli-cli-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/vrooli/binaryfetch v0.0.0 // indirect
	github.com/vrooli/platform-go v0.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	golang.org/x/sync v0.23.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/freshness-go => ../../../packages/freshness-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.11.4 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/containerd/containerd v1.7.15 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/docker v25.0.5+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shirou/gopsutil/v3 v3.23.12 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/mod v0.41.0
	golang.org/x/sys v0.48.0
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.79.3 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli => ../../..

replace intent-go => ../../../packages/intent-go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace agent-manager => ../../agent-manager/api

replace agent-manager/cli => ../../agent-manager/cli

replace device-control => ../../device-control/api

replace device-studio => ../../device-studio/api

replace device-studio/cli => ../../device-studio/cli

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/browser-automation-studio => ../../browser-automation-studio/api

replace github.com/vrooli/mdns-go => ../../../packages/mdns-go

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/pyenv-go => ../../../packages/pyenv-go

replace github.com/vrooli/scenarioconfig-go => ../../../packages/scenarioconfig-go

replace github.com/vrooli/searchregister-go => ../../../packages/searchregister-go

replace github.com/vrooli/ssh-core => ../../../packages/ssh-core

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

replace test-genie/cli => ../cli

replace ui-health => ../../ui-health/api

replace unit-health => ../../unit-health/api

replace unit-health/cli => ../../unit-health/cli

replace vrooli-autoheal-langrecover => ../../vrooli-autoheal/langrecover

replace web-console => ../../web-console/api

replace workflow-health => ../../workflow-health/api

replace workspace-sandbox => ../../workspace-sandbox/api
