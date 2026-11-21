/**
 * Mock builders for `api/generation` — the UI ↔ API generation boundary.
 * Co-located with the generation feature; deleting `features/generation/` takes
 * these mocks with it. Canonical usage:
 *
 *   import { makeGenerationMocks } from "./mocks/generation";
 *
 *   vi.mock("../../api/generation", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/generation")>();
 *     return { ...actual, ...makeGenerationMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeImageBackendStatusResponse, makeProviderStatusResponse } from "./factories";

export interface GenerationMocks {
  generationClient: {
    getProviderStatus: ReturnType<typeof vi.fn>;
    getImageBackendStatus: ReturnType<typeof vi.fn>;
  };
}

export const makeGenerationMocks = (): GenerationMocks => ({
  generationClient: {
    getProviderStatus: vi.fn().mockResolvedValue(makeProviderStatusResponse()),
    getImageBackendStatus: vi.fn().mockResolvedValue(makeImageBackendStatusResponse()),
  },
});
