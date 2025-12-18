module core-debugger

go 1.21

toolchain go1.21.13

require (
	github.com/gorilla/mux v1.8.0
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
)

replace github.com/vrooli/api-core => ../../../packages/api-core
