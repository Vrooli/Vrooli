/**
 * Mock builders for `api/apply` — the UI ↔ API apply boundary. Co-located with
 * the apply feature; deleting `features/apply/` takes these mocks with it.
 * Canonical usage:
 *
 *   import { makeApplyMocks } from "./mocks/apply";
 *
 *   vi.mock("../../api/apply", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/apply")>();
 *     return { ...actual, ...makeApplyMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeApplyResponse } from "./factories";

export interface ApplyMocks {
  previewApply: ReturnType<typeof vi.fn>;
}

export const makeApplyMocks = (): ApplyMocks => ({
  previewApply: vi.fn().mockResolvedValue(makeApplyResponse()),
});
