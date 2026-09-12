import { createClient } from "@connectrpc/connect";
import type {
  AssetRelationships,
  AssetPortContract,
  CatalogStructure,
  GetAssetRelationshipsResponse,
} from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";
import { CatalogService } from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";

import { transport } from "./client";

const client = createClient(CatalogService, transport);

export type {
  AssetRelationships,
  AssetPortContract,
  CatalogStructure,
  GetAssetRelationshipsResponse,
};

export async function getAssetRelationships(assetId: string): Promise<AssetRelationships | null> {
  const response = await client.getAssetRelationships({ assetId });
  return response.relationships ?? null;
}

export async function getAssetPortContract(assetId: string): Promise<AssetPortContract | null> {
  const response = await client.getAssetPortContract({ assetId });
  return response.contract ?? null;
}

export async function getCatalogStructure(): Promise<CatalogStructure | null> {
  const response = await client.getCatalogStructure({});
  return response.structure ?? null;
}
