# Cloud Production Health

This descriptor-backed phase runs the owner tests for scenario-to-cloud TLS and health alerts, persisted recovery, scheduled observation, DBM target plan compilation/reconciliation/readback, CLI plan/apply/wait behavior, and LPBS sign-in delivery-health provider feedback.

The assessment stores each fixed command, exit status, bounded output, duration, and owner test reference in the durable Test Genie phase result. These are **local contract results**. Fake DBM commands, event publishers, and mail-provider feedback do not prove Bridge target reach, Notification Hub delivery, SMTP/provider delivery, remote backup objects, or a staging restore.

The phase intentionally remains failed and keeps its maturity at local-contract level until CL2 staging receipts prove the current build, sign-in mail and health/TLS alert delivery, scheduled off-host backup verification, and restore drill. Production proof is another gate: after the staging work, the operator approves A1; the reviewed redeploy must retain a healthy production observation, current `build_identity`, and a certificate with more than 60 days remaining. No production operation is initiated by this phase.
