import { createClient } from "@connectrpc/connect";
import { ArticleIntakeService } from "@vrooli/proto-types/content-desk/v1/intake/intake_pb";
import { transport } from "./client";

export const intakeClient = createClient(ArticleIntakeService, transport);
