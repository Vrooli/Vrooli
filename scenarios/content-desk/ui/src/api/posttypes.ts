import { createClient } from "@connectrpc/connect";
import { PosttypesService } from "@vrooli/proto-types/content-desk/v1/posttypes/posttypes_pb";
import { transport } from "./client";

export const posttypesClient = createClient(PosttypesService, transport);
