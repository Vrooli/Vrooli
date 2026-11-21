package runtimeapp

func init() {
	auditedWriterSites["internal/setup/assets"] = map[string]string{
		"plan.go:downloadBounded:os.OpenFile#1":         "exclusive staging file for checksum metadata and image downloads; images are verified against signed hashes and recorded in ImageManifest and Receipt",
		"plan.go:writeManifestAtomic:os.CreateTemp#1":   "atomic staging of ImageManifest JSON for the scenario QEMU image manifest; ImageManifest root and schema version fact",
		"plan.go:writePrivilegedAtomic:os.CreateTemp#1": "local staging of Receipt JSON before privileged installation at the plan receipt path; Receipt root and version fact",
	}
	auditedWriterSites["internal/tools/triposr-source"] = map[string]string{
		"adapter.go:applySourceAdapter:os.WriteFile#1": "patch staged system.py in managed adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:applySourceAdapter:os.WriteFile#2": "patch staged utils.py in managed adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:applySourceAdapter:os.WriteFile#3": "patch staged isosurface.py in managed adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:applySourceAdapter:os.WriteFile#4": "write staged config adapter code in managed adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:applySourceAdapter:os.WriteFile#5": "write staged adapter notice in managed adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:copyTree:os.OpenFile#1":            "copy upstream files into staged adapter tree; computed tree digest is recorded in Receipt",
		"adapter.go:patchFile:os.WriteFile#1":          "replace checked staged adapter source file; computed tree digest is recorded in Receipt",
		"profile.go:Provision:os.WriteFile#1":          "write source-profile.json Receipt in staged managed source installation; Receipt root and schema facts",
		"profile.go:writeAtomic:os.CreateTemp#1":       "atomically stage updated source-profile.json Receipt after adapter installation; Receipt root and schema facts",
	}
}
