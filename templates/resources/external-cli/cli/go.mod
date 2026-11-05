module {{RESOURCE_CLI_COMMAND}}/cli

go 1.22

require github.com/vrooli/cli-core v0.0.0

require (
	connectrpc.com/connect v1.19.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/vrooli/cli-core => {{PACKAGES_REL_FROM_CLI}}/cli-core

replace github.com/vrooli/repo-contract-go => {{PACKAGES_REL_FROM_CLI}}/repo-contract-go

replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_CLI}}

replace github.com/vrooli/envkit-go => {{PACKAGES_REL_FROM_CLI}}/envkit-go

replace github.com/vrooli/vrooli/packages/proto => {{PACKAGES_REL_FROM_CLI}}/proto

replace github.com/vrooli/platform-go => {{PACKAGES_REL_FROM_CLI}}/platform-go
