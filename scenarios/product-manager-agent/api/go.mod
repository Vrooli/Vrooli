module product-manager-api

go 1.25.0

require (
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.20.1
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	github.com/vrooli/vrooli/packages/proto v0.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
