package schemas

import _ "embed"

//go:embed flow.schema.json
var Flow []byte

//go:embed formal-artifact.schema.json
var FormalArtifact []byte
