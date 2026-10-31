package quint

// Command names for the quint CLI subcommands flow-verifier invokes.
// These are stamped into formal artifacts and used to look up the
// arg-list for each verification step.
const (
	CommandTypecheck = "typecheck"
	CommandTest      = "test"
	CommandVerify    = "verify"
	CommandRun       = "run"
)

// TempITFPattern is the placeholder substituted in the run command's
// --out-itf argument before invocation. Quint writes one ITF file per
// trace; flow-verifier post-processes them.
const TempITFPattern = "<temp-itf-pattern>"

// VerificationBackendApalache is the backend name stamped into the
// formal artifact's source.verificationBackend field.
const VerificationBackendApalache = "apalache"

// GeneratedCheckTransitionTable is the name of the generated run-check
// emitted into every flow's Quint model — it asserts the full transition
// table matches the contract.
const GeneratedCheckTransitionTable = "transitionTable"

// QuintReservedIdentifiers are identifiers the generated Quint model
// uses internally. Contract states, events, and invariants must not
// collide with them.
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
