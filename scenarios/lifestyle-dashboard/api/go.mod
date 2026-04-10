module lifestyle-dashboard

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/vrooli/api-core v0.0.0
)

require github.com/felixge/httpsnoop v1.0.3 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core
