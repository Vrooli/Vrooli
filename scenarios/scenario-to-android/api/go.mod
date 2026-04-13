module github.com/vrooli/vrooli/scenarios/scenario-to-android/api

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go
