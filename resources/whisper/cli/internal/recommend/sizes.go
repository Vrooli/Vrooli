// Package recommend converts a HostCapabilities snapshot into a
// concrete Whisper model-size pick. The decision table is centralised
// in this package; the CLI and capacity-management paths both consume the
// same Pick() function so behaviour stays consistent.
package recommend

// Model is the canonical Whisper model identifier emitted by Pick.
// These model labels are the resource's canonical recommendation vocabulary.
type Model string

const (
	ModelTiny    Model = "tiny"
	ModelBase    Model = "base"
	ModelSmall   Model = "small"
	ModelMedium  Model = "medium"
	ModelLargeV3 Model = "large-v3"
)

// VRAMRequirement is the minimum effective VRAM (bytes) for a model at
// safe operating headroom on native Whisper runtimes. Numbers are approximate
// floor values that include activation + KV cache; they bias toward
// "the host can actually load this." Updating these is the only knob
// for shifting the table.
var VRAMRequirement = map[Model]uint64{
	ModelTiny:    1 << 30,         // 1 GB
	ModelBase:    1<<30 + 512<<20, // 1.5 GB
	ModelSmall:   2 << 30,         // 2 GB
	ModelMedium:  5 << 30,         // 5 GB
	ModelLargeV3: 10 << 30,        // 10 GB
}

// CPURAMRequirement is the minimum *budgeted* system RAM (bytes) the
// recommender requires when no GPU is present. Models above 'medium' are
// not picked on CPU even with abundant RAM — quality gain doesn't justify
// the >10× realtime factor.
var CPURAMRequirement = map[Model]uint64{
	ModelTiny:   2 << 30,
	ModelBase:   4 << 30,
	ModelSmall:  8 << 30,
	ModelMedium: 16 << 30,
}

// CPUCoreRequirement is the minimum core count for each CPU-tier pick.
var CPUCoreRequirement = map[Model]int{
	ModelTiny:   1,
	ModelBase:   2,
	ModelSmall:  4,
	ModelMedium: 8,
}
