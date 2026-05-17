module github.com/vrooli/vrooli/scenarios/audio-tools/clients/go

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/stretchr/testify v1.11.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/api-core => ../../../../packages/api-core

replace github.com/vrooli/vrooli/packages/proto => ../../../../packages/proto
