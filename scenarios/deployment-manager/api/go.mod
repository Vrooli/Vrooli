module deployment-manager

go 1.24.0

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.0
	github.com/stretchr/testify v1.11.1
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
	software.sslmate.com/src/go-pkcs12 v0.5.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..
