import { createClient } from "@connectrpc/connect";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  TunnelService,
  type TunnelStatus,
  type MetricsSample,
  type GetStatusResponse,
} from "@vrooli/proto-types/tunnel-manager/v1/tunnel/tunnel_pb";

import { transport } from "./client";

// tunnelClient is the generated Connect-Web client for TunnelService —
// composite tunnel health + Prometheus metrics time-series. Backs the
// Overview header and the Metrics surface.
export const tunnelClient = createClient(TunnelService, transport);

/** getStatus returns the composite tunnel health and latest metrics sample. */
export async function getStatus(): Promise<GetStatusResponse> {
  return tunnelClient.getStatus({});
}

/** listMetrics returns scraped metrics samples within an optional window. */
export async function listMetrics(from?: Date, to?: Date): Promise<MetricsSample[]> {
  const resp = await tunnelClient.listMetrics({
    from: from ? timestampFromDate(from) : undefined,
    to: to ? timestampFromDate(to) : undefined,
  });
  return resp.samples;
}

export type { TunnelStatus, MetricsSample, GetStatusResponse };
