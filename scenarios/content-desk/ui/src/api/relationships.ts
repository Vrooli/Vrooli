import { createClient } from "@connectrpc/connect";
import { RelationshipsService } from "@vrooli/proto-types/content-desk/v1/relationships/relationships_pb";
import { transport } from "./client";

export const relationshipsClient = createClient(RelationshipsService, transport);
