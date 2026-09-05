module github.com/vrooli/vrooli-cli-go

go 1.25.0

require (
	github.com/vrooli/repo-contract-go v0.0.0
	github.com/vrooli/vrooli/packages/proto v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/vrooli/vrooli/packages/proto => ../proto

replace github.com/vrooli/platform-go => ../platform-go

require github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect

replace github.com/vrooli/repo-contract-go => ../repo-contract-go

replace github.com/vrooli/vrooli => ../..
