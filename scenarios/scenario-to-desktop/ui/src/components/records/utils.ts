import type { DesktopRecordItemView } from "./recordPresentation";

export function pathLabel(record: DesktopRecordItemView): string {
  const rec = record.record;
  if (rec.location_mode === "temp" || rec.location_mode === "staging") {
    return rec.output_path || rec.staging_path || "staging";
  }
  return rec.output_path || rec.destination_path || "unknown";
}
