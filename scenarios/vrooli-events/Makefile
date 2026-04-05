SCENARIO_NAME := $(notdir $(CURDIR))

.PHONY: build test lint fmt start stop logs clean

build:
	cd api && GOWORK=off go build -o $(SCENARIO_NAME)-api .
	cd cli && GOWORK=off go build -o $(SCENARIO_NAME) .

test:
	vrooli scenario test $(SCENARIO_NAME)

test-go:
	GOWORK=off go test ./... -timeout 300s

lint:
	cd api && GOWORK=off golangci-lint run
	cd cli && GOWORK=off golangci-lint run

fmt:
	gofumpt -w .

start:
	vrooli scenario start $(SCENARIO_NAME)

stop:
	vrooli scenario stop $(SCENARIO_NAME)

logs:
	vrooli scenario logs $(SCENARIO_NAME)

clean:
	rm -f api/$(SCENARIO_NAME)-api cli/$(SCENARIO_NAME)
