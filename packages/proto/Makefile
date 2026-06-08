BUF_BASE_BRANCH ?= master
DESCRIPTOR_DIR ?= gen/descriptor

.PHONY: generate descriptor lint format breaking check clean refresh-vendor

generate: descriptor
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
# plugin. image.binpb is committed (the canonical artifact); image.json is an
# optional human/debug rendering and is gitignored.
descriptor:
	@mkdir -p $(DESCRIPTOR_DIR)
	buf build -o $(DESCRIPTOR_DIR)/image.binpb
	buf build -o $(DESCRIPTOR_DIR)/image.json

lint:
	buf lint

format:
	buf format -w

breaking:
	@if ! git show-ref --quiet refs/heads/$(BUF_BASE_BRANCH); then \
		echo "buf breaking baseline branch '$(BUF_BASE_BRANCH)' not found" >&2; exit 1; \
	fi
	buf breaking --against ".git#branch=$(BUF_BASE_BRANCH),subdir=packages/proto"

check: lint
	buf generate
	@mkdir -p $(DESCRIPTOR_DIR)
	buf build -o $(DESCRIPTOR_DIR)/image.binpb
	@if ! git diff --quiet gen/; then \
		echo "ERROR: Generated code (or descriptor) is out of date. Run 'make generate' and commit.'"; \
		git diff --stat gen/; \
		exit 1; \
	fi

clean:
	rm -rf gen/*

# refresh-vendor re-exports the BSR module deps (googleapis, protovalidate)
# into vendor/ so codegen continues to resolve transitive imports without
# contacting BSR. This is the single code path that requires BSR access;
# see docs/configuration/integrations/buf-bsr.md for the optional login.
# After refresh, run `make check` to confirm gen/ remains consistent.
refresh-vendor:
	buf export buf.build/googleapis/googleapis -o vendor/googleapis
	buf export buf.build/bufbuild/protovalidate -o vendor/protovalidate
