/**
 * Mock builders for `api/design` — the UI ↔ API design boundary. Co-located
 * with the design feature; deleting `features/design/` takes these mocks with
 * it. Canonical usage:
 *
 *   import { makeDesignMocks } from "./mocks/design";
 *
 *   vi.mock("../../api/design", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/design")>();
 *     return { ...actual, ...makeDesignMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeDesignResponse } from "./factories";

export interface DesignMocks {
  generateDesignLanguage: ReturnType<typeof vi.fn>;
}

export const makeDesignMocks = (): DesignMocks => ({
  generateDesignLanguage: vi.fn().mockResolvedValue(makeDesignResponse()),
});
