# Cloud deployment

The supported operator route for a hosted deployment is one Deployment Manager
operation in the phone-facing product flow. It reports setup, progress, health,
and recovery status. The operator may complete steps on the provider's own
website when the product asks for them.

Do not use shell, SSH, firewall, Bridge onboarding, or manual TLS instructions
to prepare a cloud host. The end-to-end hosted flow is still being built; this
page does not claim that staging or production deployment is currently
available. See the [Deployment Hub](../../../docs/deployment/README.md) for
current support and the [internal release governance reference](internal/RELEASE-GOVERNANCE-REFERENCE.md)
for engineering contracts and historical command examples.

For desktop deployment, use the maintained
[scenario-to-desktop documentation](../../scenario-to-desktop/docs/OVERVIEW.md).
