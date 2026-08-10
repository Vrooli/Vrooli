module github.com/vrooli/api-core

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/go-chi/chi/v5 v5.0.11
	github.com/gorilla/mux v1.8.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	golang.org/x/sys v0.42.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.1
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace github.com/vrooli/cli-core => ../cli-core

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..

replace github.com/vrooli/vrooli/packages/proto => ../proto
