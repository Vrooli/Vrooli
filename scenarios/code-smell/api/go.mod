module github.com/vrooli/code-smell/api

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core
