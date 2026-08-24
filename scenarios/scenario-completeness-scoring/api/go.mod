module scenario-completeness-scoring

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/measures-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require github.com/vrooli/maturity-go v0.0.0

require (
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/ai-go v0.0.0 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-chi/chi/v5 v5.0.11 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/vrooli/freshness-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/freshness-go => ../../../packages/freshness-go

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go
