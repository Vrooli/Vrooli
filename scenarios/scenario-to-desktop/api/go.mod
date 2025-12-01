module scenario-to-desktop-api

go 1.21

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	scenario-to-desktop-runtime v0.0.0
)

require github.com/felixge/httpsnoop v1.0.3 // indirect

replace scenario-to-desktop-runtime => ../runtime
