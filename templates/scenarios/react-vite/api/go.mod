module {{SCENARIO_ID}}

go 1.25.0

require (
	connectrpc.com/connect v1.19.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/measures-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.0
)

replace github.com/vrooli/api-core => {{PACKAGES_REL_FROM_API}}/api-core

replace github.com/vrooli/measures-go => {{PACKAGES_REL_FROM_API}}/measures-go

replace github.com/vrooli/aisearch-go => {{PACKAGES_REL_FROM_API}}/aisearch-go

replace github.com/vrooli/vrooli/packages/proto => {{PACKAGES_REL_FROM_API}}/proto

replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_API}}

replace github.com/vrooli/repo-contract-go => {{PACKAGES_REL_FROM_API}}/repo-contract-go
