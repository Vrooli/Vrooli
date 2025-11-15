# Tier 5 · Enterprise / Appliance

Tier 5 is the hardware/server line vision: customers receive a dedicated Vrooli box (or self-hosted installation) that runs all required scenarios/resources with minimal intervention.

## Principles

- **Autonomy**: The appliance must install, upgrade, and heal itself without developer involvement.
- **Compound Intelligence**: Every shipped appliance becomes another node feeding data back (when allowed) to improve future deployments.
- **Compliance & Isolation**: Secrets, data retention, audit trails, and air-gapped options must be first-class.

## State of Play

- Hardware plans are conceptual; no reference implementation exists.
- The Tier 1 stack is the closest analog, minus manufacturing/distribution concerns.
- We lack deployment metadata for hardware requirements (power, rack/tower, GPU SKUs, etc.).

## Roadmap Concepts

1. **Deployment Blueprints**: Pre-defined combinations of scenarios/resources sized for small business, enterprise, or household appliances.
2. **Provisioning Pipeline**: `deployment-manager` should emit ISO images or provisioning scripts tailored to on-prem hardware.
3. **Secrets & Identity**: Integration with secrets-manager plus enterprise identity providers (SAML/OIDC) for user onboarding.
4. **Update Channels**: OTA updates signed and distributed through Vrooli's lifecycle system.
5. **Telemetry & Support**: Optional remote support hooks that align with privacy/compliance rules.

## Next Steps

- Capture enterprise-specific constraints in deployment metadata (compliance requirements, offline mandates, HSM needs).
- Partner with hardware design efforts to define reference builds.
- Keep Tier 5 documentation synchronized with Tier 4 automation so we do not fork the code path.
