module scenario-to-desktop-api

go 1.24.0

toolchain go1.24.11

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/stretchr/testify v1.11.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	scenario-to-desktop-runtime v0.0.0
	software.sslmate.com/src/go-pkcs12 v0.4.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251213004720-97cd9d5aeac2 // indirect
)

replace scenario-to-desktop-runtime => ../runtime

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
