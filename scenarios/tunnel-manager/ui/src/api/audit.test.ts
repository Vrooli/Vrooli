import { describe, expect, it, vi } from "vitest";
import { auditClient, runAudit } from "./audit";

describe("audit API helpers", () => {
  it("returns the generated audit response", async () => {
    const response = { findings: [] };
    const spy = vi.spyOn(auditClient, "runAudit").mockResolvedValueOnce(response as never);
    await expect(runAudit()).resolves.toBe(response);
    expect(spy).toHaveBeenCalledWith({});
  });
});
