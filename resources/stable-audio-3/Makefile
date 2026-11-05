.PHONY: fmt lint type test check
fmt:
	gofmt -w cli/*.go
lint:
	go vet ./cli/...
type:
	go test -run '^$$' ./cli/...
test:
	go test ./cli/...
	cd server && PYTHONDONTWRITEBYTECODE=1 CUDA_VISIBLE_DEVICES='' python3 -m unittest -v test_server
check: lint type test
