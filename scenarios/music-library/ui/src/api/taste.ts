import { createClient } from "@connectrpc/connect";
import { CatalogService } from "@vrooli/proto-types/music-library/v1/catalog/catalog_pb";
import { TasteService } from "@vrooli/proto-types/music-library/v1/taste/taste_pb";
import { transport } from "./client";

export const tasteClient = createClient(TasteService, transport);
export const catalogClient = createClient(CatalogService, transport);
