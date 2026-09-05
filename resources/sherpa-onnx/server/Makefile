.PHONY: test test-native bundle-native bundle-native-linux-amd64 bundle-native-linux-arm64 bundle-native-macos-arm64 bundle-native-windows-amd64 build-native-linux-amd64 build-native-linux-arm64 build-native-macos-arm64 build-native-windows-amd64

ROOT ?= $(SHERPA_ONNX_ROOT)
GO ?= go

test:
	GOWORK=off $(GO) test ./...

test-native:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib" LD_LIBRARY_PATH="$(ROOT)/lib" GOWORK=off $(GO) test -tags sherpa_onnx ./...

build-native-linux-amd64:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	@test "$(HOST_GOOS)" = "linux" -a "$(HOST_GOARCH)" = "amd64" || (echo "native Linux amd64 build requires host linux/amd64, found $(HOST_GOOS)/$(HOST_GOARCH)" >&2; exit 1)
	mkdir -p dist
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib" GOWORK=off $(GO) build -trimpath -buildvcs=false -tags sherpa_onnx -o dist/sherpa-onnx-server_linux_amd64 .

build-native-linux-arm64:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	@test "$(HOST_GOOS)" = "linux" -a "$(HOST_GOARCH)" = "arm64" || (echo "native Linux arm64 build requires host linux/arm64, found $(HOST_GOOS)/$(HOST_GOARCH)" >&2; exit 1)
	mkdir -p dist
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib" GOWORK=off $(GO) build -trimpath -buildvcs=false -tags sherpa_onnx -o dist/sherpa-onnx-server_linux_arm64 .

# These targets are intentionally native-toolchain targets. The sherpa-onnx
# runtime contains platform-specific shared libraries, so a Linux host must
# not pretend it can produce a runnable macOS or Windows cgo artifact merely by
# changing GOOS/GOARCH. The host checks below make a mistaken cross-build fail
# before it emits an artifact that could be mistaken for release evidence.
build-native-macos-arm64:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	@test "$(HOST_GOOS)" = "darwin" -a "$(HOST_GOARCH)" = "arm64" || (echo "native macOS arm64 build requires host darwin/arm64, found $(HOST_GOOS)/$(HOST_GOARCH)" >&2; exit 1)
	mkdir -p dist
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib" CGO_ENABLED=1 GOWORK=off $(GO) build -trimpath -buildvcs=false -tags sherpa_onnx -o dist/sherpa-onnx-server_darwin_arm64 .

build-native-windows-amd64:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	@test "$(HOST_GOOS)" = "windows" -a "$(HOST_GOARCH)" = "amd64" || (echo "native Windows amd64 build requires host windows/amd64, found $(HOST_GOOS)/$(HOST_GOARCH)" >&2; exit 1)
	mkdir -p dist
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib" CGO_ENABLED=1 GOWORK=off $(GO) build -trimpath -buildvcs=false -tags sherpa_onnx -o dist/sherpa-onnx-server_windows_amd64.exe .

# A cgo executable is not a managed-service artifact by itself: sherpa-onnx
# and ONNX Runtime are shared libraries. These bundle targets are intentionally
# native-host only and produce the directory layout consumed by the managed
# service driver:
#
#   <bundle>/server/sherpa-onnx-server
#   <bundle>/lib/{libsherpa-onnx-c-api,libonnxruntime,...}
#
# Release publication can checksum the complete directory tree. Keeping the
# runtime libraries inside that tree prevents a host installation from
# accidentally loading an unrelated system version.
HOST_GOOS := $(shell $(GO) env GOOS)
HOST_GOARCH := $(shell $(GO) env GOARCH)

bundle-native-linux-amd64:
	$(MAKE) bundle-native PLATFORM=linux-amd64 GOOS=linux GOARCH=amd64 RPATH_FLAGS='-Wl,-rpath,\$$$$ORIGIN/../lib'

bundle-native-linux-arm64:
	$(MAKE) bundle-native PLATFORM=linux-arm64 GOOS=linux GOARCH=arm64 RPATH_FLAGS='-Wl,-rpath,\$$$$ORIGIN/../lib'

bundle-native-macos-arm64:
	$(MAKE) bundle-native PLATFORM=macos-arm64 GOOS=darwin GOARCH=arm64 RPATH_FLAGS='-Wl,-rpath,@loader_path/../lib'

bundle-native-windows-amd64:
	$(MAKE) bundle-native PLATFORM=windows-amd64 GOOS=windows GOARCH=amd64 RPATH_FLAGS=''

bundle-native:
	@test -n "$(ROOT)" || (echo 'SHERPA_ONNX_ROOT is required' >&2; exit 1)
	@test -n "$(OUT)" || (echo 'OUT is required' >&2; exit 1)
	@test "$(HOST_GOOS)" = "$(GOOS)" -a "$(HOST_GOARCH)" = "$(GOARCH)" || (echo "native bundle requires host $(GOOS)/$(GOARCH), found $(HOST_GOOS)/$(HOST_GOARCH)" >&2; exit 1)
	@test -d "$(ROOT)/include" -a -d "$(ROOT)/lib" || (echo 'SHERPA_ONNX_ROOT must contain include/ and lib/' >&2; exit 1)
	mkdir -p "$(OUT)/server" "$(OUT)/lib"
	CGO_CFLAGS="-I$(ROOT)/include" CGO_LDFLAGS="-L$(ROOT)/lib $(RPATH_FLAGS)" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=off $(GO) build -trimpath -buildvcs=false -tags sherpa_onnx -o "$(OUT)/server/sherpa-onnx-server" .
	cp -a "$(ROOT)/lib/." "$(OUT)/lib/"
	@if [ "$(GOOS)" != "windows" ]; then chmod 0755 "$(OUT)/server/sherpa-onnx-server"; fi
