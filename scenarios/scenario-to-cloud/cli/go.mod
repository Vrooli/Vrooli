module scenario-to-cloud/cli

go 1.24.0

require github.com/vrooli/cli-core v0.1.0

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go
