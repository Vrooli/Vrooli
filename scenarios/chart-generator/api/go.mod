module chart-generator-api

go 1.25.0

require (
	github.com/go-echarts/go-echarts/v2 v2.6.3
	github.com/gorilla/mux v1.8.1
	github.com/johnfercher/maroto/v2 v2.3.1
	github.com/lib/pq v1.10.9
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
)

require github.com/kr/text v0.2.0 // indirect

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/f-amaral/go-async v0.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/tiff v1.0.1 // indirect
	github.com/johnfercher/go-tree v1.0.5 // indirect
	github.com/jung-kurt/gofpdf v1.16.2 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/pdfcpu/pdfcpu v0.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/vrooli/cli-core v0.0.0 // indirect
	github.com/vrooli/repo-contract-go v0.0.0 // indirect
	golang.org/x/image v0.18.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core

replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go

replace github.com/vrooli/vrooli => ../../..

replace github.com/vrooli/binaryfetch => ../../../packages/binaryfetch

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
