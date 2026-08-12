module vrooli-bridge/agent

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/creack/pty/v2 v2.0.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..
