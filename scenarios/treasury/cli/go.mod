module treasury/cli

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	golang.org/x/crypto v0.48.0
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/envkit-go => ../../../packages/envkit-go
