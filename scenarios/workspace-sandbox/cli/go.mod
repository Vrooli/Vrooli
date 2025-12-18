module workspace-sandbox/cli

go 1.24.0

toolchain go1.24.11

require github.com/vrooli/cli-core v0.0.0

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core
