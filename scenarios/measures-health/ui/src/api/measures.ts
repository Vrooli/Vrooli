import { createClient } from "@connectrpc/connect";
import { MeasuresService } from "@vrooli/proto-types/measures-health/v1/measures/measures_pb";
import {
  TimeWindowToken,
  type TimeWindow,
} from "@vrooli/proto-types/measures/v1/measures_pb";

import { transport } from "./client";

/**
 * Connect-Web client for measures-health's own `MeasuresService` — the two
 * declared analytical measures over its `validation_run` history:
 * `countFailedValidations` ("how many scenarios failed measures validation")
 * and `countValidationCoverage` ("how many passed"). Both are read-only and
 * scoped to a canonical {@link TimeWindow}.
 */
export const measuresClient = createClient(MeasuresService, transport);

/**
 * The window tokens the card offers, in display order. These are the
 * *relative* ranges from the shared `TimeWindowToken` enum (UNSPECIFIED is the
 * oneof's "unset" sentinel and is deliberately excluded). The first entry is
 * the default selection.
 */
export const WINDOW_TOKENS = [
  TimeWindowToken.THIS_WEEK,
  TimeWindowToken.LAST_7D,
  TimeWindowToken.LAST_30D,
  TimeWindowToken.THIS_MONTH,
  TimeWindowToken.LAST_MONTH,
  TimeWindowToken.THIS_QUARTER,
] as const;

export type WindowToken = (typeof WINDOW_TOKENS)[number];

export const DEFAULT_WINDOW_TOKEN: WindowToken = TimeWindowToken.THIS_WEEK;

/** Build the shared `TimeWindow` oneof from a relative token. */
function windowFromToken(token: WindowToken): TimeWindow {
  return { window: { case: "token", value: token } } as TimeWindow;
}

/** Count scenarios that *failed* measures validation in the given window. */
export async function countFailed(token: WindowToken): Promise<bigint> {
  const res = await measuresClient.countFailedValidations({ window: windowFromToken(token) });
  return res.count;
}

/** Count scenarios that *passed* measures validation in the given window. */
export async function countCoverage(token: WindowToken): Promise<bigint> {
  const res = await measuresClient.countValidationCoverage({ window: windowFromToken(token) });
  return res.count;
}

export { TimeWindowToken };
export type { TimeWindow };
