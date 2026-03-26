module swarm-manager/cli

go 1.22

require github.com/vrooli/cli-core v0.0.0
require swarm-manager v0.0.0

replace github.com/vrooli/cli-core => ../../../packages/cli-core
replace swarm-manager => ../api
