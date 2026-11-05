module github.com/vrooli/vrooli/resources

go 1.26.0

require (
	github.com/lib/pq v1.12.3
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/vrooli/agentharness v0.0.0
	github.com/vrooli/ai-provider-go v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
	golang.org/x/sys v0.48.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	connectrpc.com/connect v1.19.2 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mikelolasagasti/xz v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/api-core v0.0.0 // indirect
	github.com/vrooli/binaryfetch v0.0.0 // indirect
	github.com/vrooli/cliresolve v0.0.0 // indirect
	github.com/vrooli/platform-go v0.0.0 // indirect
	github.com/vrooli/vrooli/packages/proto v0.0.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.50.1 // indirect
)

replace github.com/vrooli/agentharness => ../packages/agentharness

replace github.com/vrooli/api-core => ../packages/api-core

replace github.com/vrooli/binaryfetch => ../packages/binaryfetch

replace github.com/vrooli/cli-core => ../packages/cli-core

replace github.com/vrooli/cliresolve => ../packages/cliresolve

replace github.com/vrooli/envkit-go => ../packages/envkit-go

replace github.com/vrooli/platform-go => ../packages/platform-go

replace github.com/vrooli/repo-contract-go => ../packages/repo-contract-go

replace github.com/vrooli/vrooli => ..

replace github.com/vrooli/vrooli/packages/proto => ../packages/proto

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../scenarios/scenario-to-desktop/runtime

replace github.com/vrooli/ai-provider-go => ../packages/ai-provider-go
