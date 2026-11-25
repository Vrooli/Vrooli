import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { SafetyFeedbackStats } from "../../api/safetyFeedback";
import { useSafetyFeedbackStats } from "../safety/useSafetyFeedback";

/** False-positive rate as a percentage, or an em dash when no decisive label exists. */
function rate(falsePositives: bigint, confirmed: bigint, value: number): string {
  if (falsePositives + confirmed === 0n) return "—";
  return `${(value * 100).toFixed(1)}%`;
}


function StatsBody({ stats }: { stats: SafetyFeedbackStats }) {
  const { t } = useTranslation();
  const total = Number(stats.total);
  if (total === 0) {
    return (
      <p data-testid={selectors.responsibleUse.feedbackEmpty} className="text-sm text-app-muted-foreground">
        {t(strings.settings.safetyFeedback.empty)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <dl data-testid={selectors.responsibleUse.feedbackTotals} className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
        <div className="rounded-control border border-app-border p-2">
          <dt className="text-xs text-app-muted-foreground">{t(strings.settings.safetyFeedback.totalLabel)}</dt>
          <dd className="font-semibold text-app-foreground">{total}</dd>
        </div>
        <div className="rounded-control border border-app-border p-2">
          <dt className="text-xs text-app-muted-foreground">{t(strings.settings.safetyFeedback.falsePositives)}</dt>
          <dd className="font-semibold text-app-danger">{Number(stats.falsePositives)}</dd>
        </div>
        <div className="rounded-control border border-app-border p-2">
          <dt className="text-xs text-app-muted-foreground">{t(strings.settings.safetyFeedback.confirmed)}</dt>
          <dd className="font-semibold text-app-success">{Number(stats.confirmed)}</dd>
        </div>
        <div className="rounded-control border border-app-border p-2">
          <dt className="text-xs text-app-muted-foreground">{t(strings.settings.safetyFeedback.uncertain)}</dt>
          <dd className="font-semibold text-app-foreground">{Number(stats.uncertain)}</dd>
        </div>
      </dl>

      <p data-testid={selectors.responsibleUse.feedbackRate} className="text-sm text-app-foreground">
        {stats.falsePositives + stats.confirmed === 0n
          ? t(strings.settings.safetyFeedback.rateUnavailable)
          : t(strings.settings.safetyFeedback.rate, {
              rate: rate(stats.falsePositives, stats.confirmed, stats.falsePositiveRate),
            })}
      </p>

      {stats.byReason.length > 0 ? (
        <div className="flex flex-col gap-1">
          <h4 className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.settings.safetyFeedback.byReason)}
          </h4>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-app-muted-foreground">
                <th className="py-1 font-medium">{t(strings.settings.safetyFeedback.reason)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.cases)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.falsePositiveShort)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.rateColumn)}</th>
              </tr>
            </thead>
            <tbody>
              {stats.byReason.map((row) => (
                <tr key={row.reasonCode} className="border-t border-app-border">
                  <td className="py-1 font-mono text-xs text-app-foreground">{row.reasonCode || "—"}</td>
                  <td className="py-1 text-right text-app-muted-foreground">{Number(row.cases)}</td>
                  <td className="py-1 text-right text-app-muted-foreground">{Number(row.falsePositives)}</td>
                  <td className="py-1 text-right text-app-foreground">
                    {rate(row.falsePositives, row.confirmed, row.falsePositiveRate)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {stats.byProfile.length > 0 ? (
        <div className="flex flex-col gap-1">
          <h4 className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.settings.safetyFeedback.byProfile)}
          </h4>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-app-muted-foreground">
                <th className="py-1 font-medium">{t(strings.settings.safetyFeedback.profile)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.cases)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.falsePositiveShort)}</th>
                <th className="py-1 text-right font-medium">{t(strings.settings.safetyFeedback.rateColumn)}</th>
              </tr>
            </thead>
            <tbody>
              {stats.byProfile.map((row) => (
                <tr key={row.profileName || "(default)"} className="border-t border-app-border">
                  <td className="py-1 font-mono text-xs text-app-foreground">{row.profileName || "—"}</td>
                  <td className="py-1 text-right text-app-muted-foreground">{Number(row.cases)}</td>
                  <td className="py-1 text-right text-app-muted-foreground">{Number(row.falsePositives)}</td>
                  <td className="py-1 text-right text-app-foreground">
                    {rate(row.falsePositives, row.confirmed, row.falsePositiveRate)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <p className="text-xs text-app-muted-foreground">{t(strings.settings.safetyFeedback.hint)}</p>
    </div>
  );
}

/**
 * Read-only calibration panel: how often the reviewer's refusals/flags were
 * judged wrong, split by reason code and reviewer profile. The labels come from
 * the Activity control (or `image-tools safety feedback`); the corpus export is
 * the handoff to `scripts/safety-corpus-replay.py`.
 */
export function SafetyFeedbackStatsPanel() {
  const { t } = useTranslation();
  const query = useSafetyFeedbackStats();

  return (
    <section
      data-testid={selectors.responsibleUse.feedbackPanel}
      aria-labelledby="reviewer-feedback-heading"
      className="flex flex-col gap-3"
    >
      <h3 id="reviewer-feedback-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.settings.safetyFeedback.heading)}
      </h3>
      <p className="text-sm text-app-muted-foreground">{t(strings.settings.safetyFeedback.description)}</p>
      {query.isLoading ? (
        <p data-testid={selectors.responsibleUse.feedbackLoading} className="text-sm text-app-muted-foreground">
          {t(strings.settings.safetyFeedback.loading)}
        </p>
      ) : query.error || !query.data ? (
        <p data-testid={selectors.responsibleUse.feedbackError} className="text-sm text-app-danger">
          {t(strings.settings.safetyFeedback.error)}
        </p>
      ) : (
        <StatsBody stats={query.data} />
      )}
    </section>
  );
}
