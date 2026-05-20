module github.com/vrooli/api-core

go 1.24.0

require (
	github.com/go-chi/chi/v5 v5.0.11
	github.com/gorilla/mux v1.8.1
	github.com/vrooli/cli-core v0.0.0
	github.com/vrooli/repo-contract-go v0.0.0
)

require github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect

replace github.com/vrooli/cli-core => ../cli-core

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..
