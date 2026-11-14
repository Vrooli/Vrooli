module web-console

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/creack/pty/v2 v2.0.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/binaryfetch v0.0.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/platform-go v0.0.0
	github.com/vrooli/vrooli v3.2.0+incompatible
	github.com/vrooli/vrooli/packages/capability-registry-go v0.0.0
	github.com/vrooli/vrooli/packages/capabilityprobe v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	github.com/vrooli/vrooli/scenarios/audio-tools/clients/go v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/vrooli/repo-contract-go v0.0.0
	golang.org/x/sys v0.48.0
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace github.com/vrooli/platform-go => ../../../packages/platform-go

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/mikelolasagasti/xz v1.0.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/cliresolve v0.0.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/mod v0.41.0 // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/vrooli/packages/capability-registry-go => ../../../packages/capability-registry-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli/packages/capabilityprobe => ../../../packages/capabilityprobe

replace github.com/vrooli/vrooli/scenarios/audio-tools/clients/go => ../../audio-tools/clients/go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go
