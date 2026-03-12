module scenario-stack-governor

go 1.25.0

require (
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/vrooli/api-core v0.0.0
	golang.org/x/mod v0.34.0
	golang.org/x/sync v0.20.0
)

require github.com/felixge/httpsnoop v1.0.3 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core
