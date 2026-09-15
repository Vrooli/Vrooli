// Package dockerhostfirewall owns the internal safeguards docker-host-firewall boundary in Vrooli's control plane. It does not own host remediation or behavior outside this boundary; callers use its exported contracts and the owning service for those concerns.
package dockerhostfirewall
