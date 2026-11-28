.PHONY: fmt lint type test check

fmt:
	gofmt -w cli/*.go

lint:
	cd cli && go vet ./...

type:
	cd cli && go test -run '^$$' ./...

test:
	cd cli && go test ./...

check: lint type test

