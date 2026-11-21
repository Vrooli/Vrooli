import { createClient, type Client } from "@connectrpc/connect";
import { AnalyticsService } from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

import { transport } from "./client";

export const analyticsClient: Client<typeof AnalyticsService> = createClient(AnalyticsService, transport);
