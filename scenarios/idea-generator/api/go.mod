module idea-generator-api

go 1.21

require (
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
)

replace github.com/vrooli/api-core => ../../../packages/api-core
