/**
 * Test data factories for the brands domain. Co-located with the feature so
 * deleting `features/brands/` takes the factories with it (no central residue).
 *
 * The Brand / ListBrandsResponse types are GENERATED proto messages; factories
 * use `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  BrandSchema,
  ColorsSchema,
  CreateBrandResponseSchema,
  ListBrandsResponseSchema,
  type Brand,
  type Colors,
  type CreateBrandResponse,
  type ListBrandsResponse,
} from "@vrooli/proto-types/brand-manager/v1/brands/brands_pb";

export type { Brand, Colors, CreateBrandResponse, ListBrandsResponse };

// makeColors builds a full Colors message. Nested message fields must be
// constructed (not plain objects) so the proto $typeName brand is present.
export const makeColors = (overrides: MessageInitShape<typeof ColorsSchema> = {}): Colors =>
  create(ColorsSchema, { primary: "#3366ff", ...overrides });

export const makeBrand = (overrides: MessageInitShape<typeof BrandSchema> = {}): Brand =>
  create(BrandSchema, {
    id: "brand-1",
    name: "First brand",
    version: 1,
    colors: makeColors(),
    createdAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    updatedAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    ...overrides,
  });

export const makeListBrandsResponse = (
  overrides: MessageInitShape<typeof ListBrandsResponseSchema> = {},
): ListBrandsResponse =>
  create(ListBrandsResponseSchema, {
    brands: [],
    ...overrides,
  });

export const makeCreateBrandResponse = (
  overrides: MessageInitShape<typeof CreateBrandResponseSchema> = {},
): CreateBrandResponse =>
  create(CreateBrandResponseSchema, {
    brand: makeBrand(),
    ...overrides,
  });
