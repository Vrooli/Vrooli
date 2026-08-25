module image-tools

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/disintegration/imaging v1.6.2
	github.com/dsoprea/go-exif/v3 v3.0.1
	github.com/gen2brain/avif v0.4.4
	github.com/gen2brain/heic v0.4.9
	github.com/gen2brain/webp v0.5.5
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/binaryfetch v0.0.0
	github.com/vrooli/pyenv-go v0.0.0
	github.com/vrooli/vrooli-cli-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	golang.org/x/image v0.42.0
	golang.org/x/sys v0.47.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsoprea/go-logging v0.0.0-20200710184922-b02d349568dd // indirect
	github.com/dsoprea/go-utility/v2 v2.0.0-20221003172846-a3e1774ef349 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-chi/chi/v5 v5.0.11 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	golang.org/x/net v0.0.0-20221002022538-bcab6841153b // indirect
	golang.org/x/text v0.38.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/measures-go => ../../../packages/measures-go

replace github.com/vrooli/vrooli-cli-go => ../../../packages/vrooli-cli-go

replace github.com/vrooli/ai-go => ../../../packages/ai-go

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/pyenv-go => ../../../packages/pyenv-go

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go

replace github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime => ../../scenario-to-desktop/runtime
