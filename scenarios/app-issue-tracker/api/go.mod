module app-issue-tracker-api

go 1.24.0

toolchain go1.24.9

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.3
	github.com/vrooli/api-core v0.0.0
	golang.org/x/text v0.30.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/vrooli/api-core => ../../../packages/api-core
