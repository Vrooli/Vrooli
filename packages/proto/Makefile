BUF_BASE_BRANCH ?= master

.PHONY: generate lint format breaking check clean

generate:
	buf generate
	touch gen/python/py.typed gen/python/browser_automation_studio/py.typed gen/python/common/py.typed gen/python/landing_page_react_vite/py.typed

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
	@if ! git diff --quiet gen/; then \
		echo "ERROR: Generated code is out of date. Run 'make generate' and commit.'"; \
		git diff --stat gen/; \
		exit 1; \
	fi

clean:
	rm -rf gen/*
