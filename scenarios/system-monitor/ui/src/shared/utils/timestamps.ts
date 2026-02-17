/**
 * Shared timestamp utilities for the system-monitor UI.
 *
 * Extracted from useSystemMonitor to centralise proto-Timestamp helpers.
 */

import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { timestampDate } from '@bufbuild/protobuf/wkt';

/** Convert a protobuf Timestamp to an ISO-8601 string, defaulting to now. */
export const toIsoString = (ts?: Timestamp): string =>
  ts ? timestampDate(ts).toISOString() : new Date().toISOString();

/** Sort an array descending by a millisecond timestamp extracted via `getTsMs`. */
export const sortByTimestamp = <T>(items: T[], getTsMs: (item: T) => number): T[] =>
  [...items].sort((a, b) => {
    const aTime = getTsMs(a);
    const bTime = getTsMs(b);
    const aValid = Number.isFinite(aTime);
    const bValid = Number.isFinite(bTime);
    if (!aValid && !bValid) return 0;
    if (!bValid) return -1;
    if (!aValid) return 1;
    return bTime - aTime;
  });
