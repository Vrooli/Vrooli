module test-scenario

go 1.21

// No external dependencies - this is a simple test scenario
require github.com/vrooli/api-core v0.0.0

replace github.com/vrooli/api-core => ../../../packages/api-core
