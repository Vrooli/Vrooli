import { createClient, type Client } from "@connectrpc/connect";
import { CampaignService } from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

import { transport } from "./client";

export const campaignClient: Client<typeof CampaignService> = createClient(CampaignService, transport);
