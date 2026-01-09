module prd-control-tower

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
	github.com/yuin/goldmark v1.7.1
)

replace github.com/vrooli/api-core => ../../../packages/api-core
