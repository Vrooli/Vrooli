.PHONY: generate gen-code descriptor gen-manifests lint format breaking verify-committed-gen clean refresh-vendor

SCENARIO_ARGS := $(foreach scenario,$(SCENARIO),--scenario $(scenario))

generate:
	GOFLAGS=-mod=mod go run ./cmd/protogen generate $(SCENARIO_ARGS)

gen-code:
	GOFLAGS=-mod=mod go run ./cmd/protogen gen-code $(SCENARIO_ARGS)

descriptor:
	GOFLAGS=-mod=mod go run ./cmd/protogen descriptor

gen-manifests:
	GOFLAGS=-mod=mod go run ./cmd/protogen gen-manifests $(SCENARIO_ARGS)

lint:
	GOFLAGS=-mod=mod go run ./cmd/protogen lint

format:
	GOFLAGS=-mod=mod go run ./cmd/protogen format

breaking:
	GOFLAGS=-mod=mod go run ./cmd/protogen breaking --scenario $(SCENARIO)

verify-committed-gen:
	GOFLAGS=-mod=mod go run ./cmd/protogen verify

clean:
	GOFLAGS=-mod=mod go run ./cmd/protogen clean

refresh-vendor:
	GOFLAGS=-mod=mod go run ./cmd/protogen refresh-vendor
