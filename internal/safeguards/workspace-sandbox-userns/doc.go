// Package workspacesandboxuserns owns the internal safeguards workspace-sandbox-userns boundary in Vrooli's control plane. It does not own host remediation or behavior outside this boundary; callers use its exported contracts and the owning service for those concerns.
package workspacesandboxuserns
