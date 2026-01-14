module workspace-sandbox

go 1.24.0

toolchain go1.24.11

require (
	github.com/creack/pty/v2 v2.0.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/felixge/httpsnoop v1.0.3 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
