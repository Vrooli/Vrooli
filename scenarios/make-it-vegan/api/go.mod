module make-it-vegan

go 1.21

require (
	github.com/gorilla/mux v1.8.0
	github.com/rs/cors v1.8.2
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.14.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core
