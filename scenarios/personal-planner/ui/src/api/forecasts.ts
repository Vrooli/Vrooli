import { createClient } from "@connectrpc/connect";
import { ForecastsService, type Forecast, type ForecastSnapshotSummary } from "@vrooli/proto-types/personal-planner/v1/forecasts/forecasts_pb";

import { transport } from "./client";

const forecastsClient = createClient(ForecastsService, transport);
export type { Forecast, ForecastSnapshotSummary };

export async function fetchForecast(input: { localDate?: string; timezone?: string; horizonDays?: number } = {}): Promise<Forecast> {
  const response = await forecastsClient.getForecast({ localDate: input.localDate ?? "", timezone: input.timezone ?? "", horizonDays: input.horizonDays ?? 28 });
  if (!response.forecast) throw new Error("The server returned no forecast");
  return response.forecast;
}

export async function fetchForecastHistory(limit = 8): Promise<ForecastSnapshotSummary[]> {
  const response = await forecastsClient.listForecastSnapshots({ limit });
  return response.snapshots;
}
