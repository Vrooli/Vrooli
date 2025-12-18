module document-manager-api

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.7.0
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
