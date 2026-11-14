import { createClient } from "@connectrpc/connect";
import { ModelSelectionService } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";

import { transport } from "./client";

export const modelSelectionClient = createClient(ModelSelectionService, transport);
