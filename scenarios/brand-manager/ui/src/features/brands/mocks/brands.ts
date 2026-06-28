/**
 * Mock builders for `api/brands` — the UI ↔ API brands-CRUD boundary.
 * Co-located with the brands feature; deleting `features/brands/` takes these
 * mocks with it. Canonical usage:
 *
 *   import { makeBrandsMocks } from "./mocks/brands";
 *
 *   vi.mock("../../api/brands", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/brands")>();
 *     return { ...actual, ...makeBrandsMocks() };
 *   });
 */
import { vi } from "vitest";

import { makeBrand, makeCreateBrandResponse, makeListBrandsResponse } from "./factories";

export interface BrandsMockCreateInput {
  name: string;
  description?: string;
}

export interface BrandsMocks {
  brandsClient: {
    listBrands: ReturnType<typeof vi.fn>;
    createBrand: ReturnType<typeof vi.fn>;
    getBrand: ReturnType<typeof vi.fn>;
    listBrandVersions: ReturnType<typeof vi.fn>;
  };
}

export const makeBrandsMocks = (): BrandsMocks => ({
  brandsClient: {
    listBrands: vi.fn().mockResolvedValue(makeListBrandsResponse()),
    createBrand: vi
      .fn()
      .mockImplementation((input: BrandsMockCreateInput) =>
        Promise.resolve(makeCreateBrandResponse({ brand: makeBrand({ name: input.name }) })),
      ),
    getBrand: vi
      .fn()
      .mockImplementation((input: { id: string }) => Promise.resolve({ brand: makeBrand({ id: input.id }) })),
    listBrandVersions: vi.fn().mockResolvedValue({ versions: [] }),
  },
});
