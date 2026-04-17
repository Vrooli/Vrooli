module secrets-manager-api

go 1.22

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go
