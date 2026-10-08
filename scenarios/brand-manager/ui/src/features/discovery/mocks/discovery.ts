/**
 * Mock builders for `api/discovery` — the UI ↔ API discovery boundary.
 * Co-located with the discovery feature; deleting `features/discovery/` takes
 * these mocks with it. Canonical usage:
 *
 *   import { makeDiscoveryMocks } from "./mocks/discovery";
 *
 *   vi.mock("../../api/discovery", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/discovery")>();
 *     return { ...actual, ...makeDiscoveryMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeDiscoveryResult } from "./factories";

export interface DiscoveryMocks {
  discoverScenario: ReturnType<typeof vi.fn>;
}

export const makeDiscoveryMocks = (): DiscoveryMocks => ({
  discoverScenario: vi.fn().mockResolvedValue(makeDiscoveryResult()),
});
