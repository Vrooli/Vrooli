import { Badge } from "../../../components/ui/badge";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { type ExperimentRow } from "../../../services/experiment";

export function StatusLabel({ status }: { status: ExperimentRow["status"] }) {
  const { t } = useTranslation();
  switch (status) {
    case "queued":
      return <>{t(strings.dictationStudio.statusQueued)}</>;
    case "running":
      return <>{t(strings.dictationStudio.statusRunning)}</>;
    case "succeeded":
      return <>{t(strings.dictationStudio.statusSucceeded)}</>;
    case "failed":
      return <>{t(strings.dictationStudio.statusFailed)}</>;
    case "canceled":
      return <>{t(strings.dictationStudio.statusCanceled)}</>;
    default:
      return <>{t(strings.dictationStudio.statusUnspecified)}</>;
  }
}

export function StatusBadge({ status }: { status: ExperimentRow["status"] }) {
  const variant = status === "succeeded" ? "success" : status === "failed" || status === "canceled" ? "danger" : status === "running" ? "info" : "neutral";
  return (
    <Badge variant={variant}>
      <StatusLabel status={status} />
    </Badge>
  );
}

export function StrategyName({ kind }: { kind: string }) {
  const { t } = useTranslation();
  switch (kind) {
    case "batch":
      return <>{t(strings.dictationStudio.strategyBatch)}</>;
    case "vad_segment":
      return <>{t(strings.dictationStudio.strategyVadSegment)}</>;
    case "overlap_agree":
      return <>{t(strings.dictationStudio.strategyOverlapAgree)}</>;
    default:
      return <>{kind}</>;
  }
}
