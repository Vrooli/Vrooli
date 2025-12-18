module github.com/vrooli/scenarios/token-economy/api

go 1.21

require (
	github.com/google/uuid v1.5.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.3.0
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
