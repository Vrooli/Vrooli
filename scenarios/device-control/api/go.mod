module device-control

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/godbus/dbus/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/jezek/xgb v1.3.0
	github.com/stretchr/testify v1.12.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/mdns-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli v3.2.0+incompatible
	github.com/vrooli/vrooli/packages/capability-registry-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	golang.org/x/sys v0.47.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-chi/chi/v5 v5.0.11 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pion/datachannel v1.5.9 // indirect
	github.com/pion/dtls/v3 v3.0.3 // indirect
	github.com/pion/ice/v4 v4.0.2 // indirect
	github.com/pion/interceptor v0.1.37 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns/v2 v2.0.7 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.14 // indirect
	github.com/pion/rtp v1.8.9 // indirect
	github.com/pion/sctp v1.8.33 // indirect
	github.com/pion/sdp/v3 v3.0.9 // indirect
	github.com/pion/srtp/v3 v3.0.4 // indirect
	github.com/pion/stun/v3 v3.0.0 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	github.com/pion/turn/v4 v4.0.0 // indirect
	github.com/pion/webrtc/v4 v4.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/binaryfetch v0.0.0 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/platform-go v0.0.0 // indirect
	github.com/wlynxg/anet v0.0.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/vrooli/packages/capability-registry-go => ../../../packages/capability-registry-go

replace github.com/vrooli/mdns-go => ../../../packages/mdns-go

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve
