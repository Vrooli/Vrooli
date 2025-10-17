module financial-calculators-hub

go 1.21

replace financial-calculators-hub/lib => ../lib

require (
	financial-calculators-hub/lib v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/jung-kurt/gofpdf v1.16.2
	github.com/lib/pq v1.10.9
)
