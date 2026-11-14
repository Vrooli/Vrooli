.PHONY: fmt lint test
fmt:
	gofmt -w cli/*.go
lint:
	go vet ./cli/...
test:
	go test ./cli/...
	cd server && PYTHONDONTWRITEBYTECODE=1 CUDA_VISIBLE_DEVICES='' python3 -m unittest -v
