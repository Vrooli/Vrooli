module web-scraper-manager-api

go 1.24

toolchain go1.24.7

require (
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/chromedp/cdproto v0.0.0-20250724212937-08a3db8b4327
	github.com/chromedp/chromedp v0.14.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20250725192818-e39067aee2d2 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
