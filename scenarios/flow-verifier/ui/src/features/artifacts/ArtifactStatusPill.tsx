// ArtifactStatusPill is the shared visual that signals whether a flow's
// generated/ tree is ready. Three states map to three tones:
//
//   - "fresh"           → green "Fresh"
//   - "missing"         → yellow "Needs generate"
//   - "needs_generate"  → yellow alias used when the signal comes from
//                          a RunRow's failureReason rather than a live
//                          artifacts status fetch
//
// Used by Inventory rows, Flow Detail, Run Detail, and scenario tables.
import { useTranslation } from "../../i18n";

export type PillStatus = "fresh" | "missing" | "needs_generate";

interface Props {
  status: PillStatus;
  testId?: string;
}

function tone(status: PillStatus): string {
  switch (status) {
    case "fresh":
      return "bg-app-success/15 text-app-success";
    case "missing":
    case "needs_generate":
      return "bg-app-warning/20 text-app-warning";
  }
}

function label(status: PillStatus, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (status) {
    case "fresh":
      return t("artifacts.statusFresh", { defaultValue: "Fresh" });
    case "missing":
      return t("artifacts.statusMissing", { defaultValue: "Needs generate" });
    case "needs_generate":
      return t("artifacts.statusNeedsGenerate", { defaultValue: "Needs generate" });
  }
}

export function ArtifactStatusPill({ status, testId }: Props) {
  const { t } = useTranslation();
  return (
    <span
      data-testid={testId ?? `artifact-status-${status}`}
      className={`inline-flex items-center rounded-pill px-2.5 py-0.5 text-xs font-medium ${tone(status)}`}
    >
      {label(status, t)}
    </span>
  );
}
