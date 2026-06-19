BUF_BASE_BRANCH ?= master
DESCRIPTOR_DIR ?= gen/descriptor

.PHONY: generate gen-code descriptor gen-manifests lint format breaking verify-committed-gen clean refresh-vendor

generate: gen-code descriptor gen-manifests

gen-code:
	go run ./cmd/prune-gen
	buf generate
	# Mark every generated Python package directory as typed (PEP 561).
	# Recursive find replaces the previous hardcoded list so adding a new
	# scenario doesn't require Makefile edits.
	@if [ -d gen/python ]; then \
		find gen/python -type d -exec touch {}/py.typed \; ; \
	fi

# descriptor emits a self-contained FileDescriptorSet ("image") so request-
# message param schemas — buf.validate constraints, enum values, optionality,
# and leading comments — are runtime-introspectable by packages/measures-go and
# any other consumer. This is a `buf build` invocation, not a `buf generate`
# plugin. image.binpb is committed as the canonical descriptor artifact.
descriptor:
	@mkdir -p $(DESCRIPTOR_DIR)
	buf build -o $(DESCRIPTOR_DIR)/image.binpb

gen-manifests:
	rm -rf gen/manifests
	go run ./cmd/gen-manifest

lint:
	buf lint

format:
	buf format -w

breaking:
	@if [ -n "$(SCENARIO)" ]; then \
		proto-health impact scenario "$(SCENARIO)"; \
	else \
		for dir in schemas/*; do \
			if [ -d "$$dir" ]; then \
				proto-health impact scenario "$$(basename "$$dir")"; \
			fi; \
		done; \
	fi

verify-committed-gen: lint generate
	@if ! git diff --quiet gen/; then \
		echo "ERROR: Committed generated code (or descriptor) does not match schemas. Run 'make generate' and commit the generated artifacts.'"; \
		git diff --stat gen/; \
		exit 1; \
	fi

clean:
	rm -rf gen/*

# refresh-vendor re-exports the BSR module deps (googleapis, protovalidate)
# into vendor/ so codegen continues to resolve transitive imports without
# contacting BSR. This is the single code path that requires BSR access;
# see docs/configuration/integrations/buf-bsr.md for the optional login.
# After refresh, run `make verify-committed-gen` to confirm committed gen/
# remains consistent.
refresh-vendor:
	buf export buf.build/googleapis/googleapis -o vendor/googleapis
	buf export buf.build/bufbuild/protovalidate -o vendor/protovalidate
