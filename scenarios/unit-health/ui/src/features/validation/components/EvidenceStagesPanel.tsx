import type { EvidenceStages } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

const known = (value: string | undefined, allowed: readonly string[]) => value && allowed.includes(value) ? value : "unknown";

export function EvidenceStagesPanel({ stages }: { stages?: EvidenceStages }) {
  const { t } = useTranslation();
  const rows = [
    ["configured", known(stages?.configured, ["observed", "cached"])],
    ["analyzed", known(stages?.analyzed, ["observed", "partial", "cached"])],
    ["executed", known(stages?.executed, ["not_requested", "not_executed", "passed", "failed", "refused", "cached"])],
    ["reviewed", known(stages?.reviewed, ["not_supplied"])],
  ];
  return <section aria-label={t(strings.validation.evidenceStagesTitle)} className="rounded-panel border border-app-border bg-app-surface p-4">
    <h3 className="text-sm font-semibold">{t(strings.validation.evidenceStagesTitle)}</h3>
    <p className="mt-2 text-sm">{t(strings.validation.evidenceStagesLimit)}</p>
    <dl className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
      {rows.map(([name, state]) => <div key={name}><dt className="text-xs text-app-muted-foreground">{name}</dt><dd className="text-sm">{state}</dd></div>)}
    </dl>
    {stages?.sourceRunId && <code className="mt-2 block break-words text-xs">{`source_run=${stages.sourceRunId}`}</code>}
  </section>;
}
