module fall-foliage-explorer

go 1.25.0

require (
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..
