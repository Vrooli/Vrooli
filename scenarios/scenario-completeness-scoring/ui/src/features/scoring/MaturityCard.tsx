import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { MaturityHeadline } from "../../api/scoring";

interface MaturityCardProps {
  maturity: MaturityHeadline;
  /** Current tree digest ("td:…") the snapshot was computed against. */
  digest: string;
  /** Reason the digest is unavailable; rendered when `digest` is empty. */
  digestError: string;
}

/**
 * Maturity headline: working rung / ladder-clean state, build status, the
 * honesty label ("as of digest td:…"), and per-dimension finding counts.
 * Mirrors the CLI report's MATURITY section.
 */
export function MaturityCard({ maturity, digest, digestError }: MaturityCardProps) {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.scoring.maturity.card}
      aria-label={t(strings.scoring.maturity.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.maturity.title)}
      </h3>
      <dl className="mt-2 space-y-1 text-sm">
        {maturity.ladderClean ? (
          <div data-testid={selectors.scoring.maturity.ladderClean} className="font-medium text-emerald-600 dark:text-emerald-400">
            {t(strings.scoring.maturity.ladderClean)}
          </div>
        ) : (
          <>
            <div className="flex gap-2">
              <dt className="text-app-muted-foreground">{t(strings.scoring.maturity.workingRungLabel)}</dt>
              <dd data-testid={selectors.scoring.maturity.workingRung} className="font-medium">
                {maturity.workingRung || "R0"}
              </dd>
            </div>
            <div className="flex gap-2">
              <dt className="text-app-muted-foreground">{t(strings.scoring.maturity.satisfiedThroughLabel)}</dt>
              <dd data-testid={selectors.scoring.maturity.satisfiedThrough}>
                {maturity.satisfiedThrough || t(strings.scoring.maturity.none)}
              </dd>
            </div>
          </>
        )}
        <div className="flex gap-2">
          <dt className="text-app-muted-foreground">{t(strings.scoring.maturity.buildLabel)}</dt>
          <dd data-testid={selectors.scoring.maturity.build}>
            {maturity.buildPassing
              ? t(strings.scoring.maturity.buildPassing)
              : t(strings.scoring.maturity.buildFailing)}
          </dd>
        </div>
        <div className="flex gap-2">
          <dt className="text-app-muted-foreground">{t(strings.scoring.maturity.digestLabel)}</dt>
          <dd data-testid={selectors.scoring.maturity.digest} className="break-all font-mono text-xs">
            {digest || `${t(strings.scoring.maturity.digestUnavailable)}${digestError ? ` (${digestError})` : ""}`}
          </dd>
        </div>
      </dl>
      {maturity.dimensions.length > 0 && (
        <table data-testid={selectors.scoring.maturity.dimensions} className="mt-3 w-full text-sm">
          <thead>
            <tr className="text-start text-xs uppercase text-app-muted-foreground">
              <th scope="col" className="py-1 text-start">{t(strings.scoring.maturity.dimensionHeader)}</th>
              <th scope="col" className="py-1 text-end">{t(strings.scoring.maturity.errorPlusHeader)}</th>
              <th scope="col" className="py-1 text-end">{t(strings.scoring.maturity.openHeader)}</th>
            </tr>
          </thead>
          <tbody>
            {maturity.dimensions.map((dimension) => (
              <tr key={dimension.dimension} className="border-t border-app-border">
                <th scope="row" className="py-1 text-start font-normal">
                  {dimension.dimension}
                  {dimension.approximate && (
                    <span className="ms-1 text-xs text-app-muted-foreground">
                      ({t(strings.scoring.maturity.approximate)})
                    </span>
                  )}
                </th>
                <td className={`py-1 text-end ${dimension.errorPlus > 0 ? "font-semibold text-amber-600 dark:text-amber-400" : ""}`}>
                  {dimension.errorPlus}
                </td>
                <td className="py-1 text-end">{dimension.total}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
