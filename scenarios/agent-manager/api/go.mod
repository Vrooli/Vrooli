module agent-manager

go 1.25.0

require (
	buf.build/go/protovalidate v1.1.0
	connectrpc.com/connect v1.19.2
	github.com/google/cel-go v0.26.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/jmoiron/sqlx v1.4.0
	github.com/prometheus/client_golang v1.19.0
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.12.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/measures-go v0.0.0
	github.com/vrooli/platform-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/searchregister-go v0.0.0-00010101000000-000000000000
	github.com/vrooli/vrooli v0.0.0-00010101000000-000000000000
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	github.com/go-chi/chi/v5 v5.0.11 // indirect
	github.com/vrooli/binaryfetch v0.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	cel.dev/expr v0.24.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/vrooli/ai-go v0.0.0
	github.com/vrooli/maturity-go v0.0.0
	golang.org/x/exp v0.0.0-20250813145105-42675adae3e6 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace github.com/vrooli/searchregister-go => ../../../packages/searchregister-go
