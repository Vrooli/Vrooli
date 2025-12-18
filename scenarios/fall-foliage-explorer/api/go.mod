module fall-foliage-explorer

go 1.21

toolchain go1.21.13

require (
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0-00010101000000-000000000000
)

replace github.com/vrooli/api-core => ../../../packages/api-core
