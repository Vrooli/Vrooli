import { createClient } from "@connectrpc/connect";
import { PreparationService } from "@vrooli/proto-types/image-tools/v1/preparation/preparation_pb";

import { transport } from "./client";

export const preparationClient = createClient(PreparationService, transport);
