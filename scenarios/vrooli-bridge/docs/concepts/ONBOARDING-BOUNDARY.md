# Remote onboarding boundary

`vrooli-bridge` owns transport, pairing, and the durable remote operation. It
does not reconstruct Vrooli operator state and it does not decide which
scenario or resource declarations are needed on a target.

When a setup profile names scenarios or resources, Bridge sends a capability-
shaped selection document to the remote `vrooli-onboarding` CLI. The document
is transferred through a private temporary file; selection JSON and credential
values are never placed in command arguments. Onboarding remains the one write
authority for operator state and delegates host mutations to the control plane.

After the apply completes, Bridge invokes `vrooli-onboarding readiness --json`
on the same target. The returned metadata-safe readiness report and exit code
are authoritative. A zero exit means required items are ready; a non-zero exit
is preserved as the onboarding failure code and the report's remediation is
surfaced in the durable Bridge operation.

This narrow contract is shared by desktop and VPS provisioning: consumers use
the onboarding union export to decide which scenario, resource, host-tool, and
safeguard declarations to ship, while the target's onboarding service remains
responsible for validating the resulting host.
