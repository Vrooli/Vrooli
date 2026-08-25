import type { QualityVerdict } from "../../api/studio";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";

/**
 * The perceptual verdict, with its margins.
 *
 * A pass/fail badge alone is the wrong shape for this information. The gate's
 * job is to make "unusable" impossible to ship, not to enforce a house style,
 * so the interesting question is almost never *did it pass* — it is *by how
 * much*, because a style clearing its bar by a hair is one retune away from
 * being refused, and an operator should see that before shipping it.
 *
 * `reserved_quiet` is a ceiling rather than a floor: the reserved region must
 * be quieter than the rest of the frame, so a lower number is better. Rendering
 * every metric with the same "more is better" bar would make it read backwards.
 */
const CEILING_METRICS = new Set(["reserved_quiet"]);

export function QualityMeters({ verdict }: { verdict: QualityVerdict | null }) {
  const { t } = useTranslation();

  if (!verdict) {
    return (
      <p className="text-sm text-app-muted-foreground" data-testid="quality-absent">
        {t(strings.pages.style.qualityAbsent)}
      </p>
    );
  }

  const names = Object.keys(verdict.metrics).sort();

  return (
    <div className="flex flex-col gap-3" data-testid="quality-meters">
      <div className="flex items-center gap-2">
        <StatusBadge>
          {verdict.passed ? t(strings.pages.style.qualityPass) : t(strings.pages.style.qualityFail)}
        </StatusBadge>
        <span className="text-sm text-app-muted-foreground">
          {t(strings.pages.style.qualityCaption)}
        </span>
      </div>
      <dl className="grid gap-2 sm:grid-cols-2">
        {names.map((name) => {
          const value = verdict.metrics[name] ?? 0;
          const threshold = verdict.thresholds[name];
          const ceiling = CEILING_METRICS.has(name);
          const clears =
            threshold === undefined ? true : ceiling ? value <= threshold : value >= threshold;
          return (
            <div
              key={name}
              className="rounded-control border border-app-border p-2"
              data-testid={`quality-metric-${name}`}
              data-clears={clears ? "true" : "false"}
            >
              <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{name}</dt>
              <dd className="flex items-baseline gap-2">
                <span className="font-mono text-sm font-semibold">{value.toFixed(3)}</span>
                {threshold === undefined ? null : (
                  <span className="text-xs text-app-muted-foreground">
                    {ceiling
                      ? t(strings.pages.style.qualityCeiling, { threshold: threshold.toFixed(3) })
                      : t(strings.pages.style.qualityFloor, { threshold: threshold.toFixed(3) })}
                  </span>
                )}
              </dd>
            </div>
          );
        })}
      </dl>
      {verdict.failures && verdict.failures.length > 0 ? (
        <ul className="list-disc pl-5 text-sm" role="alert">
          {verdict.failures.map((failure) => (
            <li key={failure}>{failure}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
