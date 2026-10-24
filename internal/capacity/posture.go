package capacity

const (
	CapacityPostureResponsive = "responsive"
	CapacityPostureBalanced   = "balanced"
	CapacityPostureThroughput = "throughput"
	CapacityPostureMinimal    = "minimal"
)

type PostureResourceDefault struct {
	Priority string
	Rung     string
}

type PostureDefaults struct {
	ReserveBytes int64
	Resources    map[string]PostureResourceDefault
}

var preferredRungs = map[string]string{
	"ollama":               "gemma4:12b",
	"reranker":             "gpu",
	"whisper":              "vulkan",
	"kokoro":               "gpu",
	"kyutai-stt":           "gpu",
	"speaker-verification": "gpu",
}

var floorRungs = map[string]string{
	"ollama":               "cpu",
	"reranker":             "cpu",
	"whisper":              "cpu",
	"kokoro":               "cpu",
	"kyutai-stt":           "cpu",
	"speaker-verification": "cpu",
}

// CapacityPostures is data, not branching policy. Per-resource operator
// choices are applied after these defaults by the application layer.
var CapacityPostures = map[string]PostureDefaults{
	CapacityPostureResponsive: {
		ReserveBytes: 6 * 1024 * 1024 * 1024,
		Resources: postureResources(preferredRungs, map[string]string{
			"ollama": policyService, "reranker": policyService,
			"whisper": policyInteractive, "kokoro": policyInteractive,
			"kyutai-stt": policyInteractive, "speaker-verification": policyInteractive,
		}),
	},
	CapacityPostureBalanced: {
		ReserveBytes: 4 * 1024 * 1024 * 1024,
		Resources: postureResources(preferredRungs, map[string]string{
			"ollama": policyService, "reranker": policyService, "whisper": policyService,
			"kokoro": policyService, "kyutai-stt": policyService, "speaker-verification": policyService,
		}),
	},
	CapacityPostureThroughput: {
		ReserveBytes: 2 * 1024 * 1024 * 1024,
		Resources: postureResources(preferredRungs, map[string]string{
			"ollama": policyInteractive, "reranker": policyService, "whisper": policyService,
			"kokoro": policyService, "kyutai-stt": policyService, "speaker-verification": policyService,
		}),
	},
	CapacityPostureMinimal: {
		ReserveBytes: 8 * 1024 * 1024 * 1024,
		Resources: postureResources(floorRungs, map[string]string{
			"ollama": policyBatch, "reranker": policyBatch, "whisper": policyBatch,
			"kokoro": policyBatch, "kyutai-stt": policyBatch, "speaker-verification": policyBatch,
		}),
	},
}

func postureResources(rungs, priorities map[string]string) map[string]PostureResourceDefault {
	out := make(map[string]PostureResourceDefault, len(rungs))
	for resource, rung := range rungs {
		out[resource] = PostureResourceDefault{Rung: rung, Priority: priorities[resource]}
	}
	return out
}

func ResolvePosture(posture string) PostureDefaults {
	if defaults, ok := CapacityPostures[posture]; ok {
		return defaults
	}
	return CapacityPostures[CapacityPostureBalanced]
}

// FitStateForPosture applies the declared precedence in one place: explicit
// resource choice, then posture default, then the manifest default used by Fit
// when neither names a value.
func FitStateForPosture(posture string, reserveOverride *int64, overrides map[string]FitResourceChoice) FitState {
	defaults := ResolvePosture(posture)
	out := FitState{Posture: posture, ReserveBytes: defaults.ReserveBytes, Resources: make(map[string]FitResourceChoice)}
	if out.Posture == "" {
		out.Posture = CapacityPostureBalanced
	}
	if reserveOverride != nil {
		out.ReserveBytes = *reserveOverride
	}
	for name, resource := range defaults.Resources {
		out.Resources[name] = FitResourceChoice{Rung: resource.Rung, Priority: resource.Priority}
	}
	for name, override := range overrides {
		choice := out.Resources[name]
		choice.Enabled = override.Enabled
		if override.Rung != "" {
			choice.Rung = override.Rung
		}
		if override.Priority != "" {
			choice.Priority = override.Priority
		}
		if override.Tunables != nil {
			choice.Tunables = override.Tunables
		}
		out.Resources[name] = choice
	}
	return out
}
