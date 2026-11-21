package release

import (
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// ConfigurationDigestRule documents how the nonsecret configuration digest is
// derived; it is recorded verbatim in inputs.json so a reader can recompute.
const ConfigurationDigestRule = "sha256 over canonical JSON (sorted keys, compact) of the cloud manifest with secrets set to null and the target locator (target.vps.host, port, user, key_path, workdir) cleared; the identity-neutral remainder (scenario, dependencies, bundle, ports, edge, environment, version, preserve_paths) is the configuration"

// ConfigurationSchema names the manifest schema the digest was taken over.
func ConfigurationSchema(m domain.CloudManifest) string {
	if m.Version == "" {
		return "cloud-manifest"
	}
	return "cloud-manifest/" + m.Version
}

// NonsecretConfiguration returns the manifest with every secret and every
// target locator field removed. The locator is mutable transport
// configuration, never identity; secrets are never part of any digest.
func NonsecretConfiguration(m domain.CloudManifest) domain.CloudManifest {
	out := m
	out.Secrets = nil
	if m.Target.VPS != nil {
		vps := *m.Target.VPS
		vps.Host = ""
		vps.Port = 0
		vps.User = ""
		vps.Workdir = ""
		out.Target.VPS = &vps
	}
	return out
}

// ConfigurationDigest returns the lowercase sha256 hex of the nonsecret
// configuration.
func ConfigurationDigest(m domain.CloudManifest) (string, error) {
	return cloudrelease.CanonicalDigest(NonsecretConfiguration(m))
}
