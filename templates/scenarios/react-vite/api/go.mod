module {{SCENARIO_ID}}

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.10.0
	github.com/vrooli/api-core v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.0
)

replace github.com/vrooli/api-core => {{PACKAGES_REL_FROM_API}}/api-core

replace github.com/vrooli/vrooli/packages/proto => {{PACKAGES_REL_FROM_API}}/proto

replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_API}}
