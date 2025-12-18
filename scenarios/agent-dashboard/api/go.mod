module agent-dashboard

go 1.24.0

toolchain go1.24.7

require github.com/google/uuid v1.6.0

require (
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
	golang.org/x/time v0.13.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core
