module github.com/vrooli/job-to-scenario-pipeline

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/vrooli/api-core v0.0.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/vrooli/api-core => ../../../packages/api-core
