module brand-manager-api

go 1.21

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
)

require github.com/DATA-DOG/go-sqlmock v1.5.2

replace github.com/vrooli/api-core => ../../../packages/api-core
