package hostwireless

import "os"

// FirmwarePolicyRequest must be freshly reconstructed by the privileged owner;
// it is not an authority token. Root is an injected filesystem namespace (usually
// an already admitted host root). No mutation or subprocess is performed.
type FirmwarePolicyRequest struct {
	Root          *os.Root
	KernelRelease string
	Firmware      []FirmwarePolicyTarget
	Modules       []string // exact module basenames, without .ko; '-' and '_' are equivalent
}
type FirmwarePolicyTarget struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}
type FirmwarePolicySelection struct {
	Name          string `json:"name"`
	Path          string `json:"path,omitempty"` // root-relative, actual selected encoding
	Encoding      string `json:"encoding,omitempty"`
	SHA256        string `json:"sha256,omitempty"` // decoded bytes
	StoredSHA256  string `json:"stored_sha256,omitempty"`
	Builtin       bool   `json:"builtin"`
	StageRequired bool   `json:"stage_required"`
	StagePath     string `json:"stage_path,omitempty"`
}
type FirmwarePolicyEvidence struct {
	ParameterSHA256      string                    `json:"parameter_sha256"`
	BuiltinModulesSHA256 string                    `json:"builtin_modules_sha256"`
	KernelRelease        string                    `json:"kernel_release"`
	ConfigSHA256         string                    `json:"config_sha256"`
	CustomPath           string                    `json:"custom_path"`
	Selections           []FirmwarePolicySelection `json:"selections"`
	InitramfsSHA256      string                    `json:"initramfs_sha256"`
	InitramfsAffected    []string                  `json:"initramfs_affected"`
	BuiltinModules       []string                  `json:"builtin_modules"`
	StageAllowed         bool                      `json:"stage_allowed"`
	Reasons              []string                  `json:"reasons,omitempty"`
}
