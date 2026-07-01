module secure-document-processing-api

go 1.25.0

require (
	github.com/gorilla/mux v1.8.1
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
