package preflight

// Canonical VPS requirement values used by runtime checks and CLI/docs output.
const (
	SupportedOSID             = "ubuntu"
	RecommendedUbuntuVersion  = "24.04"
	SupportedUbuntuAltVersion = "22.04"
	LegacyUbuntuAltVersion    = "20.04"

	MinDiskFreeKB    int64 = 5 * 1024 * 1024 // 5 GiB
	MinRAMKB         int64 = 512 * 1024      // 512 MiB
	RecommendedRAMKB int64 = 2 * 1024 * 1024 // 2 GiB
	DefaultSSHPort         = 22
	DefaultHTTPPort        = 80
	DefaultHTTPSPort       = 443
)

// RequirementsResponse is returned by GET /api/v1/preflight/requirements.
type RequirementsResponse struct {
	VPS struct {
		OS struct {
			RequiredID            string   `json:"required_id"`
			RequiredFamily        string   `json:"required_family"`
			RecommendedVersion    string   `json:"recommended_version"`
			CompatibleVersions    []string `json:"compatible_versions"`
			UnsupportedBehavior   string   `json:"unsupported_behavior"`
			CompatibilityBehavior string   `json:"compatibility_behavior"`
		} `json:"os"`
		Resources struct {
			MinDiskFreeKB       int64 `json:"min_disk_free_kb"`
			MinDiskFreeBytes    int64 `json:"min_disk_free_bytes"`
			MinRAMKB            int64 `json:"min_ram_kb"`
			MinRAMBytes         int64 `json:"min_ram_bytes"`
			RecommendedRAMKB    int64 `json:"recommended_ram_kb"`
			RecommendedRAMBytes int64 `json:"recommended_ram_bytes"`
		} `json:"resources"`
		Network struct {
			RequiredInboundPorts []int `json:"required_inbound_ports"`
			SSHPort              int   `json:"ssh_port"`
		} `json:"network"`
		Authentication struct {
			RequiredMethod string `json:"required_method"`
			BootstrapFlow  string `json:"bootstrap_flow"`
		} `json:"authentication"`
	} `json:"vps"`
}

// BuildRequirementsResponse returns the canonical requirement contract.
func BuildRequirementsResponse() RequirementsResponse {
	var resp RequirementsResponse

	resp.VPS.OS.RequiredID = SupportedOSID
	resp.VPS.OS.RequiredFamily = "ubuntu-lts"
	resp.VPS.OS.RecommendedVersion = RecommendedUbuntuVersion
	resp.VPS.OS.CompatibleVersions = []string{SupportedUbuntuAltVersion, LegacyUbuntuAltVersion}
	resp.VPS.OS.UnsupportedBehavior = "hard_fail_preflight"
	resp.VPS.OS.CompatibilityBehavior = "warn_preflight"

	resp.VPS.Resources.MinDiskFreeKB = MinDiskFreeKB
	resp.VPS.Resources.MinDiskFreeBytes = MinDiskFreeKB * 1024
	resp.VPS.Resources.MinRAMKB = MinRAMKB
	resp.VPS.Resources.MinRAMBytes = MinRAMKB * 1024
	resp.VPS.Resources.RecommendedRAMKB = RecommendedRAMKB
	resp.VPS.Resources.RecommendedRAMBytes = RecommendedRAMKB * 1024

	resp.VPS.Network.RequiredInboundPorts = []int{DefaultSSHPort, DefaultHTTPPort, DefaultHTTPSPort}
	resp.VPS.Network.SSHPort = DefaultSSHPort

	resp.VPS.Authentication.RequiredMethod = "ssh_key"
	resp.VPS.Authentication.BootstrapFlow = "scenario-to-cloud ssh bootstrap"

	return resp
}
