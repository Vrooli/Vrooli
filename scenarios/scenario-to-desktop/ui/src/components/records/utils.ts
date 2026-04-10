import type { DesktopRecordResponse } from "../../lib/api";

export function pathLabel(record: DesktopRecordResponse["records"][number]): string {
  const rec = record.record;
  if (!rec) return "Unknown";
  if (rec.location_mode === "temp" || rec.location_mode === "staging") {
    return rec.output_path || rec.staging_path || "staging";
  }
  return rec.output_path || rec.destination_path || "unknown";
}
