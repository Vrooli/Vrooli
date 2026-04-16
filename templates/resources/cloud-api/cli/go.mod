module {{RESOURCE_CLI_COMMAND}}/cli

go 1.22

require github.com/vrooli/cli-core v0.0.0

replace github.com/vrooli/cli-core => {{PACKAGES_REL_FROM_CLI}}/cli-core

replace github.com/vrooli/repo-contract-go => {{PACKAGES_REL_FROM_CLI}}/repo-contract-go

replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_CLI}}
