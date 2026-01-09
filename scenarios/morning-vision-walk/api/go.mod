module morning-vision-walk

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
)

require golang.org/x/net v0.17.0 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core
