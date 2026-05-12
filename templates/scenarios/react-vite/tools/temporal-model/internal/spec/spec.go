package spec

const (
	SchemaVersion    = 6
	GeneratorVersion = 7
	GeneratorPath    = "tools/temporal-model"

	VerificationBackendApalache = "apalache"

	GeneratedCheckTransitionTable = "transitionTable"
)

const (
	CommandTypecheck = "typecheck"
	CommandTest      = "test"
	CommandVerify    = "verify"
	CommandRun       = "run"
)

const TempITFPattern = "<temp-itf-pattern>"

var QuintReservedIdentifiers = map[string]bool{
	"Status":                      true,
	"Event":                       true,
	"init":                        true,
	"step":                        true,
	"apply":                       true,
	"isValid":                     true,
	"nextStatus":                  true,
	GeneratedCheckTransitionTable: true,
	"rejected":                    true,
	"status":                      true,
	"event":                       true,
}

var IgnoredDirectories = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	"_apalache-out": true,
}
