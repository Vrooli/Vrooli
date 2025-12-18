module vrooli-autoheal

go 1.24.0

toolchain go1.24.11

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	golang.org/x/text v0.31.0
)

require github.com/felixge/httpsnoop v1.0.3 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core
