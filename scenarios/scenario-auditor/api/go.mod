module scenario-auditor

go 1.21

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
)

require github.com/traefik/yaegi v0.15.1

replace github.com/vrooli/api-core => ../../../packages/api-core
