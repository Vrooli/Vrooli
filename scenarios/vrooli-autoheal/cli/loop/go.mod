module vrooli-autoheal-loop

go 1.25.0

require github.com/vrooli/repo-contract-go v0.0.0

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/vrooli/envkit-go v0.0.0
)

replace github.com/vrooli/repo-contract-go => ../../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../../..

replace github.com/vrooli/envkit-go => ../../../../packages/envkit-go

require vrooli-autoheal-langrecover v0.0.0

replace vrooli-autoheal-langrecover => ../../langrecover
