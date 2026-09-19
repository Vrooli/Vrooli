import { createClient } from "@connectrpc/connect";
import {
  BrandsService,
  type Brand,
  type BrandVersion,
  type ListBrandsResponse,
} from "@vrooli/proto-types/brand-manager/v1/brands/brands_pb";

import { transport } from "./client";

export const brandsClient = createClient(BrandsService, transport);

/** listBrands returns brands ordered newest-updated first. */
export async function listBrands(nameContains = ""): Promise<Brand[]> {
  const resp = await brandsClient.listBrands({ nameContains });
  return resp.brands;
}

/** createBrand persists a new brand (name required) and returns it at version 1. */
export async function createBrand(input: {
  name: string;
  description?: string;
}): Promise<Brand> {
  const resp = await brandsClient.createBrand({
    name: input.name,
    description: input.description ?? "",
  });
  if (!resp.brand) {
    throw new Error("create returned no brand");
  }
  return resp.brand;
}

/** getBrand fetches a single brand by id. */
export async function getBrand(id: string): Promise<Brand> {
  const resp = await brandsClient.getBrand({ id });
  if (!resp.brand) {
    throw new Error("get returned no brand");
  }
  return resp.brand;
}

/** listBrandVersions returns a brand's immutable version history, newest-first. */
export async function listBrandVersions(brandId: string): Promise<BrandVersion[]> {
  const resp = await brandsClient.listBrandVersions({ brandId });
  return resp.versions;
}

export type { Brand, BrandVersion, ListBrandsResponse };
