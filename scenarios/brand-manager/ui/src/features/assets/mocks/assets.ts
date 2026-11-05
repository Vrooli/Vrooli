/**
 * Mock builders for `api/assets` — the UI ↔ API assets-catalog boundary.
 * Co-located with the assets feature; deleting `features/assets/` takes these
 * mocks with it. Canonical usage:
 *
 *   import { makeAssetsMocks } from "./mocks/assets";
 *
 *   vi.mock("../../api/assets", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/assets")>();
 *     return { ...actual, ...makeAssetsMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeAsset, makeListAssetsResponse } from "./factories";

export interface AssetsMocks {
  assetsClient: {
    listAssets: ReturnType<typeof vi.fn>;
    getAsset: ReturnType<typeof vi.fn>;
  };
}

export const makeAssetsMocks = (): AssetsMocks => ({
  assetsClient: {
    listAssets: vi.fn().mockResolvedValue(makeListAssetsResponse()),
    getAsset: vi
      .fn()
      .mockImplementation((input: { id: string }) => Promise.resolve({ asset: makeAsset({ id: input.id }) })),
  },
});
