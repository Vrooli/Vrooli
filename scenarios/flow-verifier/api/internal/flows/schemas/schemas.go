package schemas

import _ "embed"

//go:embed temporal.schema.json
var Temporal []byte

//go:embed navigation.schema.json
var Navigation []byte

//go:embed formal-artifact.schema.json
var FormalArtifact []byte

//go:embed examples/navigation-minimal.json
var NavigationMinimalExample []byte

//go:embed examples/navigation-full.json
var NavigationFullExample []byte
