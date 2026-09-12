/**
 * Helpers for the generated proto wire types. Kept tiny and dependency-light so
 * any surface can turn an optional `google.protobuf.Timestamp` into a JS `Date`
 * (or `undefined`, which the posture views treat as a first-class "never").
 */
import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";

export function tsToDate(ts: Timestamp | undefined): Date | undefined {
  return ts ? timestampDate(ts) : undefined;
}
