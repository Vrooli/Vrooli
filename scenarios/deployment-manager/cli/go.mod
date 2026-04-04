module deployment-manager/cli

go 1.24.0

toolchain go1.24.11

require (
	github.com/vrooli/cli-core v0.1.0
	deployment-manager v0.0.0
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace deployment-manager => ../api
