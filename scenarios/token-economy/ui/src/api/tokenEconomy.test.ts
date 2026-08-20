import { describe, expect, it } from "vitest";

import { earningClient, holderClient, minterClient } from "./tokenEconomy";

describe("token economy generated clients", () => {
  it("exposes authority work only on the minter client", () => {
    expect(minterClient.createTokenType).toBeTypeOf("function");
    expect(minterClient.approveRedemption).toBeTypeOf("function");
    expect("createTokenType" in holderClient).toBe(false);
  });

  it("keeps operator entry on the ordinary earning contract", () => {
    expect(earningClient.submitEarning).toBeTypeOf("function");
  });
});

