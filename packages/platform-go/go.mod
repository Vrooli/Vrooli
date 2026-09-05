module github.com/vrooli/platform-go

go 1.25.0

require (
	github.com/vrooli/repo-contract-go v0.0.0
	golang.org/x/sys v0.42.0
)

require github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect

replace github.com/vrooli/repo-contract-go => ../repo-contract-go
