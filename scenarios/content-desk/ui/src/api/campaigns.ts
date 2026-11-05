import { createClient } from "@connectrpc/connect";
import { CampaignsService } from "@vrooli/proto-types/content-desk/v1/campaigns/campaigns_pb";
import { transport } from "./client";

export const campaignsClient = createClient(CampaignsService, transport);
