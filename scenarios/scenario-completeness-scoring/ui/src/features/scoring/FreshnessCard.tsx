import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { FreshnessBlock, PhaseFreshness } from "../../api/scoring";
import { verdictBadgeClass, verdictKey } from "./format";

interface FreshnessCardProps {
  freshness: FreshnessBlock;
}

/**
 * Per-phase freshness verdicts with the copy-pastable refresh command.
 * Mirrors the CLI report's FRESHNESS section (freshness-go semantics:
 * fresh = a passing run recorded at the current tree digest).
 */
export function FreshnessCard({ freshness }: FreshnessCardProps) {
  const { t } = useTranslation();

  const detail = (phase: PhaseFreshness): string => {
    switch (phase.verdict) {
      case "fresh":
        return t(strings.scoring.freshness.lastRun, { id: phase.lastRunId });
      case "stale":
        return phase.lastRunId
          ? t(strings.scoring.freshness.lastPassed, {
              id: phase.lastRunId,
              digest: phase.lastDigest || t(strings.scoring.freshness.unstampedDigest),
            })
          : t(strings.scoring.freshness.neverPassed);
      default:
        return t(strings.scoring.freshness.noEvidence);
    }
  };

  return (
    <section
      data-testid={selectors.scoring.freshness.card}
      aria-label={t(strings.scoring.freshness.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.freshness.title)}
      </h3>
      <ul className="mt-2 space-y-1 text-sm">
        {freshness.phases.map((phase) => (
          <li
            key={phase.phase}
            data-testid={selectors.scoring.freshnessPhaseRow({ phase: phase.phase })}
            className="flex flex-wrap items-baseline gap-x-2 border-t border-app-border py-1 first:border-t-0"
          >
            <span className="min-w-24 font-medium">{phase.phase}</span>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${verdictBadgeClass(phase.verdict)}`}>
              {t(verdictKey(phase.verdict))}
            </span>
            <span className="text-xs text-app-muted-foreground">{detail(phase)}</span>
          </li>
        ))}
      </ul>
      {freshness.suggestedCommand && (
        <div className="mt-3 text-sm">
          <span className="text-app-muted-foreground">{t(strings.scoring.freshness.refreshLabel)}</span>{" "}
          <code
            data-testid={selectors.scoring.freshness.refreshCommand}
            className="break-all rounded bg-app-background px-1.5 py-0.5 font-mono text-xs"
          >
            {freshness.suggestedCommand}
          </code>
        </div>
      )}
    </section>
  );
}
