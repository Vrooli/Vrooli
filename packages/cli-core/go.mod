module github.com/vrooli/cli-core

go 1.23

require (
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli v0.0.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..
