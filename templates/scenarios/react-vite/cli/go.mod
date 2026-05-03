module {{SCENARIO_ID}}/cli

go 1.22

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/api-core v0.0.0 // indirect
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/vrooli/api-core => {{PACKAGES_REL_FROM_CLI}}/api-core

replace github.com/vrooli/cli-core => {{PACKAGES_REL_FROM_CLI}}/cli-core

replace github.com/vrooli/vrooli/packages/proto => {{PACKAGES_REL_FROM_CLI}}/proto

replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_CLI}}
