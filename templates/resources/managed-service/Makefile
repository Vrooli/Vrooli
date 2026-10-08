.PHONY: test

# Toolchain floor (GOFLAGS -p width, GOMAXPROCS, pnpm concurrency) shared with every Makefile in the repository.
-include ../../mk/toolchain.mk

test:
	cd cli && go test ./...
