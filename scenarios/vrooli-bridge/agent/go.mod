module vrooli-bridge/agent

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/creack/pty/v2 v2.0.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/cliresolve v0.0.0
	github.com/vrooli/envkit-go v0.0.0
	github.com/vrooli/mdns-go v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go

replace github.com/vrooli/mdns-go => ../../../packages/mdns-go

replace github.com/vrooli/cliresolve => ../../../packages/cliresolve

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/platform-go => ../../../packages/platform-go
