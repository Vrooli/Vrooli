export type AkashPriceEvidence = {
  provider?: string;
  offerId?: string;
  region?: string;
  size?: string;
  currency?: string;
  serverMinorPerMonth?: bigint;
  ipv4MinorPerMonth?: bigint;
  taxBasis?: string;
  sourceUrl?: string;
  sourceVersion?: string;
  observedAt?: { seconds: bigint; nanos: number };
  expiresAt?: { seconds: bigint; nanos: number };
};

export type AkashProviderFacts = {
  billingFacts: Record<string, string>;
};

export type AkashReadinessFacts = {
  credentialStatus: string;
  purchaseEnabled: boolean;
  prices: AkashPriceEvidence[];
};

export type AkashSetupView = {
  network: string;
  effectMode: string;
  signerStatus: string;
  priceStatus: string;
  blockedReason: string;
  setupActionsEnabled: boolean;
  readOnlyProbeEnabled: boolean;
};

const observedMillis = (value?: { seconds: bigint; nanos: number }) =>
  value ? Number(value.seconds) * 1000 + value.nanos / 1_000_000 : Number.NaN;

export function akashSetupView(
  provider: AkashProviderFacts,
  readiness: AkashReadinessFacts | undefined,
  now = Date.now(),
): AkashSetupView {
  const facts = provider.billingFacts;
  const prices = readiness?.prices ?? [];
  const hasFreshSourcedPrice = prices.some((price) =>
    Boolean(price.provider === "akash" && price.offerId && price.region && price.size && price.currency && price.taxBasis && price.sourceUrl && price.sourceVersion) &&
    (price.serverMinorPerMonth ?? -1n) >= 0n &&
    (price.ipv4MinorPerMonth ?? -1n) >= 0n &&
    observedMillis(price.observedAt) <= now &&
    observedMillis(price.expiresAt) > now,
  );
  const signerVerified = facts.credential_authority_signer_status === "verified";
  const liveGate = facts.effect_mode === "live";
  const testnetOnly = facts.network === "sandbox-2";
  const backendReady = readiness?.credentialStatus === "configured" && readiness.purchaseEnabled && hasFreshSourcedPrice;
  return {
    network: facts.network || "",
    effectMode: facts.effect_mode || "",
    signerStatus: signerVerified ? "verified" : facts.credential_authority_signer_status || "unreported",
    priceStatus: hasFreshSourcedPrice ? "fresh_sourced" : prices.length ? "stale_or_invalid" : facts.price_status || "no_fact",
    blockedReason: facts.blocked_reason || "",
    setupActionsEnabled: facts.setup_actions_enabled === "true" && testnetOnly && liveGate && signerVerified && backendReady,
    readOnlyProbeEnabled: facts.read_only_probe_enabled === "true" && testnetOnly && liveGate && signerVerified && readiness?.credentialStatus === "configured",
  };
}
