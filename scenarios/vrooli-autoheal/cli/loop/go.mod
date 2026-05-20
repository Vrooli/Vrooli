module vrooli-autoheal-loop

go 1.24.0

require github.com/vrooli/repo-contract-go v0.0.0

require github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect

replace github.com/vrooli/repo-contract-go => ../../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../../..
