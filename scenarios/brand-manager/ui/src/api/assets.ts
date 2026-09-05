import { createClient } from "@connectrpc/connect";
import {
  AssetsService,
  type Asset,
  type ListAssetsResponse,
} from "@vrooli/proto-types/brand-manager/v1/assets/assets_pb";

import { transport } from "./client";

export const assetsClient = createClient(AssetsService, transport);

/** listAssets returns asset catalog entries ordered newest-uploaded first. */
export async function listAssets(brandId = ""): Promise<Asset[]> {
  const resp = await assetsClient.listAssets({ brandId });
  return resp.assets;
}

/** getAsset returns a single asset catalog entry by id. */
export async function getAsset(id: string): Promise<Asset> {
  const resp = await assetsClient.getAsset({ id });
  if (!resp.asset) {
    throw new Error("get returned no asset");
  }
  return resp.asset;
}

export type { Asset, ListAssetsResponse };
