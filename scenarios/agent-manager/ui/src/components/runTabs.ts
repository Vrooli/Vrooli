import { Activity, File, FileCode, Gauge, Info } from "lucide-react";

/** Run detail sections, in tab order; `?tab=<id>` deep-links one. */
export const RUN_TABS = [
  { id: "task", label: "Task", Icon: File },
  { id: "timeline", label: "Timeline", Icon: Activity },
  { id: "diff", label: "Diff", Icon: FileCode },
  { id: "usage", label: "Usage", Icon: Gauge },
  { id: "report", label: "Report", Icon: Info },
] as const;

export type RunTabId = (typeof RUN_TABS)[number]["id"];

export function isRunTab(value: string | null): value is RunTabId {
  return RUN_TABS.some((tab) => tab.id === value);
}
