/**
 * Test data factories for the assets domain. Co-located with the feature so
 * deleting `features/assets/` takes the factories with it (no central residue).
 *
 * The Asset / ListAssetsResponse types are GENERATED proto messages; factories
 * use `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  AssetSchema,
  ListAssetsResponseSchema,
  UploadAssetResponseSchema,
  type Asset,
  type ListAssetsResponse,
  type UploadAssetResponse,
} from "@vrooli/proto-types/brand-manager/v1/assets/assets_pb";

export type { Asset, ListAssetsResponse, UploadAssetResponse };

export const makeAsset = (overrides: MessageInitShape<typeof AssetSchema> = {}): Asset =>
  create(AssetSchema, {
    id: "asset-1",
    brandId: "brand-1",
    filename: "logo.png",
    mimeType: "image/png",
    size: 2048n,
    createdAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    ...overrides,
  });

export const makeListAssetsResponse = (
  overrides: MessageInitShape<typeof ListAssetsResponseSchema> = {},
): ListAssetsResponse =>
  create(ListAssetsResponseSchema, {
    assets: [],
    ...overrides,
  });

export const makeUploadAssetResponse = (
  overrides: MessageInitShape<typeof UploadAssetResponseSchema> = {},
): UploadAssetResponse =>
  create(UploadAssetResponseSchema, {
    asset: makeAsset(),
    ...overrides,
  });
