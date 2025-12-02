module test-genie/cli

go 1.24.0

toolchain go1.24.11

require (
	github.com/vrooli/cli-core v0.0.0
	golang.org/x/term v0.37.0
)

require golang.org/x/sys v0.38.0 // indirect

replace github.com/vrooli/cli-core => ../../../packages/cli-core
