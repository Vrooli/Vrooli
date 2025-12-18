module secrets-manager-api

go 1.21

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	gopkg.in/yaml.v2 v2.4.0
)

require github.com/felixge/httpsnoop v1.0.1 // indirect

replace github.com/vrooli/api-core => ../../../packages/api-core
