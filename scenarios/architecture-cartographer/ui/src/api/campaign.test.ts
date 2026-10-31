import { describe, expect, it } from "vitest";

import { campaignClient } from "./campaign";

describe("api/campaign.campaignClient", () => {
  it("exposes every CampaignService RPC as a callable method", () => {
    const rpcs = [
      "createCampaign",
      "listCampaigns",
      "getCampaignStatus",
      "nextCampaignStep",
      "resolveItem",
      "applyItem",
      "reauditCampaign",
      "closeCampaign",
    ] as const;
    for (const rpc of rpcs) {
      expect(typeof campaignClient[rpc]).toBe("function");
    }
  });
});
