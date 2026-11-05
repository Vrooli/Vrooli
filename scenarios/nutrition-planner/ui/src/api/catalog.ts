import { createClient } from "@connectrpc/connect";
import { CatalogService, type CatalogRevision } from "@vrooli/proto-types/nutrition-planner/v1/catalog/catalog_pb";
import { transport } from "./client";

const client = createClient(CatalogService, transport);

export type { CatalogRevision };

export async function listCatalogRevisions(workspaceId: string): Promise<CatalogRevision[]> {
  const response = await client.listCatalog({ workspaceId });
  return response.revisions;
}
