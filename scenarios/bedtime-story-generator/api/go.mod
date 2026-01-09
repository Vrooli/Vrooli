module bedtime-story-generator

go 1.21.1

toolchain go1.21.13

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/google/uuid v1.5.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/rs/cors v1.10.1
	github.com/stretchr/testify v1.11.1
	github.com/vrooli/api-core v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jung-kurt/gofpdf v1.16.2
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vrooli/api-core => ../../../packages/api-core
