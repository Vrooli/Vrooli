import { describe, expect, it } from "vitest";
import { akashSetupView, type AkashPriceEvidence } from "./akashReadiness";

const now = Date.parse("2026-11-24T12:00:00Z");
const currentPrice: AkashPriceEvidence = {
  provider: "akash",
  offerId: "fixture-small",
  region: "testnet",
  size: "small",
  currency: "USD",
  serverMinorPerMonth: 3500n,
  ipv4MinorPerMonth: 0n,
  taxBasis: "pre-tax",
  sourceUrl: "https://prices.example.invalid",
  sourceVersion: "fixture-current",
  observedAt: { seconds: BigInt(now / 1000 - 60), nanos: 0 },
  expiresAt: { seconds: BigInt(now / 1000 + 3600), nanos: 0 },
};

describe("Akash setup/readiness projection", () => {
  it("shows testnet, stub, missing price, and missing signer only from server facts", () => {
    const view = akashSetupView({ billingFacts: {
      network: "sandbox-2",
      effect_mode: "stub",
      price_status: "no_valid_price",
      credential_authority_signer_status: "unverified",
      blocked_reason: "testnet signer and current price facts are absent",
      setup_actions_enabled: "false",
      read_only_probe_enabled: "false",
    } }, { credentialStatus: "missing", purchaseEnabled: false, prices: [] }, now);
    expect(view).toMatchObject({
      network: "sandbox-2",
      effectMode: "stub",
      signerStatus: "unverified",
      priceStatus: "no_valid_price",
      blockedReason: "testnet signer and current price facts are absent",
      setupActionsEnabled: false,
      readOnlyProbeEnabled: false,
    });
  });

  it("does not invent missing metadata or enable actions when server facts are absent", () => {
    const view = akashSetupView({ billingFacts: {} }, undefined, now);
    expect(view.network).toBe("");
    expect(view.effectMode).toBe("");
    expect(view.signerStatus).toBe("unreported");
    expect(view.priceStatus).toBe("no_fact");
    expect(view.blockedReason).toBe("");
    expect(view.setupActionsEnabled).toBe(false);
    expect(view.readOnlyProbeEnabled).toBe(false);
  });

  it("keeps actions disabled under the stub effect mode even with verified-looking facts", () => {
    const view = akashSetupView({ billingFacts: {
      network: "sandbox-2", effect_mode: "stub", credential_authority_signer_status: "verified",
      setup_actions_enabled: "true", read_only_probe_enabled: "true",
    } }, { credentialStatus: "configured", purchaseEnabled: true, prices: [currentPrice] }, now);
    expect(view.priceStatus).toBe("fresh_sourced");
    expect(view.setupActionsEnabled).toBe(false);
    expect(view.readOnlyProbeEnabled).toBe(false);
  });

  it("requires fresh sourced price and explicit backend action gates", () => {
    const view = akashSetupView({ billingFacts: {
      network: "sandbox-2", effect_mode: "live", credential_authority_signer_status: "verified",
      setup_actions_enabled: "true", read_only_probe_enabled: "true",
    } }, { credentialStatus: "configured", purchaseEnabled: true, prices: [currentPrice] }, now);
    expect(view.setupActionsEnabled).toBe(true);
    expect(view.readOnlyProbeEnabled).toBe(true);
    const noPrice = akashSetupView({ billingFacts: {
      network: "sandbox-2", effect_mode: "live", credential_authority_signer_status: "verified",
      setup_actions_enabled: "true", read_only_probe_enabled: "true",
    } }, { credentialStatus: "configured", purchaseEnabled: true, prices: [] }, now);
    expect(noPrice.setupActionsEnabled).toBe(false);
    const stale = akashSetupView({ billingFacts: {
      network: "sandbox-2", effect_mode: "live", credential_authority_signer_status: "verified",
      setup_actions_enabled: "true", read_only_probe_enabled: "true",
    } }, { credentialStatus: "configured", purchaseEnabled: true, prices: [{ ...currentPrice, expiresAt: { seconds: BigInt(now / 1000 - 1), nanos: 0 } }] }, now);
    expect(stale.priceStatus).toBe("stale_or_invalid");
    expect(stale.setupActionsEnabled).toBe(false);
  });
});
