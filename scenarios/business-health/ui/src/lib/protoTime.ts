import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";

/**
 * Convert an optional `google.protobuf.Timestamp` (as decoded by connect) to a
 * JS `Date`, or `undefined` when the field is unset. Centralized so every
 * surface renders evidence/scan timestamps the same way instead of
 * re-deriving the `seconds`/`nanos` math.
 */
export function timestampToDate(ts: Timestamp | undefined): Date | undefined {
  if (!ts) return undefined;
  // A zero timestamp (seconds === 0n, nanos === 0) is proto3's "unset" sentinel
  // for a message field that was default-constructed; treat it as absent so the
  // UI shows "no evidence" rather than the Unix epoch.
  if (ts.seconds === 0n && ts.nanos === 0) return undefined;
  return timestampDate(ts);
}
