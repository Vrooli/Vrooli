module email-triage

go 1.24.0

require (
	github.com/emersion/go-imap v1.2.1
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/qdrant/go-client v1.7.0
	github.com/rs/cors v1.10.1
	github.com/vrooli/api-core v0.0.0
	google.golang.org/grpc v1.79.3
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
)

require (
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/google/uuid v1.6.0
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
