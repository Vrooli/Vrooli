import { describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createRecordsService } from "./records-service";

function client(): IApiClient {
  return { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
}

const draftResponse = {
  disposition: "draft",
  record: {
    id: "rec-draft-1", kind: "fix", scenario: "swarm-manager", trigger: "", approach: "",
    ruled_out: [], outcome: "", draft: true, created_at: "2026-06-10T00:00:00Z",
    capture: { raw: { kind: "fix", scenario: "swarm-manager" }, accepted: { kind: "fix" }, needs: ["outcome"], invalid: [], warnings: ["private"] },
  },
  accepted: { kind: "fix" }, needs: ["outcome"], invalid: [], warnings: ["private"],
  next_action: ["swarm-manager", "records", "edit", "--repair"],
};

describe("records service progressive capture", () => {
  it("posts permissive capture input and maps draft repair guidance", async () => {
    const api = client();
    vi.mocked(api.post).mockResolvedValue(draftResponse);

    const result = await createRecordsService(api).capture({
      kind: "fix", scenario: "swarm-manager", trigger: "", approach: "", ruledOut: [], outcome: "",
    });

    expect(api.post).toHaveBeenCalledWith("/records/capture", {
      kind: "fix", scenario: "swarm-manager", trigger: "", approach: "", evidence: "", ruled_out: [], outcome: "", created_by: "", idempotency_key: "",
    });
    expect(result).toMatchObject({ disposition: "draft", needs: ["outcome"], record: { id: "rec-draft-1", draft: true } });
  });

  it("repairs the same draft through its capture endpoint", async () => {
    const api = client();
    vi.mocked(api.patch).mockResolvedValue({ ...draftResponse, disposition: "published", record: { ...draftResponse.record, draft: false, outcome: "shipped" } });

    const result = await createRecordsService(api).repairCapture("rec-draft-1", {
      kind: "fix", scenario: "swarm-manager", trigger: "Repair", approach: "Added UI", ruledOut: [], outcome: "shipped",
    });

    expect(api.patch).toHaveBeenCalledWith("/records/rec-draft-1/capture", expect.objectContaining({ outcome: "shipped", trigger: "Repair" }));
    expect(result.disposition).toBe("published");
  });
});
