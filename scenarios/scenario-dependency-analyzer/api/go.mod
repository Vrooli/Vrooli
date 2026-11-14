module github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api

go 1.26.0

require (
	connectrpc.com/connect v1.19.2
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.6.0
	github.com/iancoleman/orderedmap v0.2.0
	github.com/joho/godotenv v1.4.0
	github.com/rs/cors v1.11.0
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/vrooli/ai-go v0.0.0
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/binaryfetch v0.0.0
	github.com/vrooli/freshness-go v0.0.0
	github.com/vrooli/maturity-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/searchregister-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
	github.com/vrooli/vrooli-cli-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require golang.org/x/mod v0.41.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/mikelolasagasti/xz v1.0.1 // indirect
	github.com/vrooli/cliresolve v0.0.0 // indirect
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/platform-go v0.0.0
	github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime v0.0.0
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/freshness-go => ../../../packages/freshness-go

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/maturity-go => ../../../packages/maturity-go

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/searchregister-go => ../../../packages/searchregister-go

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve
