module github.com/vrooli/resources/sqlite

go 1.24.0

require (
	github.com/vrooli/cli-core v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.46.0
	modernc.org/sqlite v1.40.1
)

replace github.com/vrooli/cli-core => ../../packages/cli-core

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/sys v0.39.0 // indirect
	modernc.org/libc v1.66.10 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
