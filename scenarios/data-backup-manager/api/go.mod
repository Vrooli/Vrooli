module data-backup-manager-api

go 1.24.0

require (
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
)

require github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..
