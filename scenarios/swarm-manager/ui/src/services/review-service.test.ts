import { describe, expect, it, vi } from "vitest";
import { createReviewService, type ReviewVerificationClient } from "./review-service";
import type { IApiClient } from "../lib/api-client";

describe("review service", () => {
  it("verifies evidence through the typed attempt contract", async () => {
    const apiClient = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() } as unknown as IApiClient;
    const verifyAttemptEvidence = vi.fn().mockResolvedValue({ verified: true });
    const service = createReviewService(apiClient, { verifyAttemptEvidence } as unknown as ReviewVerificationClient);

    await service.verifyEvidence("execute", "item", 2, "proof", true, "legacy-execution-id", "operator@example.test", "I inspected the artifact.");

    expect(verifyAttemptEvidence).toHaveBeenCalledWith({
      subjectKind: "backlog-item",
      subjectRef: "execute/item",
      roundNum: 2,
      evidenceId: "proof",
      verified: true,
      actor: "operator@example.test",
      reason: "I inspected the artifact.",
    });
    expect(apiClient.post).not.toHaveBeenCalled();
  });
});
