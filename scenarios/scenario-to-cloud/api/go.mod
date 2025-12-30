module scenario-to-cloud

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	golang.org/x/crypto v0.28.0
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	golang.org/x/sys v0.26.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
