import { useQuery } from "@tanstack/react-query";
import { ShieldCheck, ShieldAlert } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Severity, validationClient } from "../../api/validation";
import { compareSeverity, severityMeta } from "./severity";

/**
 * Embeddable security-posture badge — a compact severity rollup plus the top
 * findings for a single scenario, designed to drop inline into another surface
 * (an agent-inbox card, a fleet dashboard tile). It is self-contained: pass a
 * `scenario` id and it fetches + renders without any host wiring.
 *
 * @vrooliWidget widgetId=security-health-posture-badge slot=INLINE scope=SCENARIO
 * @widgetComponent PostureBadge
 * @widgetDescription Security posture severity rollup with the top findings and last-scan verdict for one scenario.
 * @widgetProps {"type":"object","properties":{"scenario":{"type":"string","description":"Scenario id to validate"},"topN":{"type":"number","description":"Max findings to surface"}},"required":["scenario"]}
 *
 * The `@vrooliWidget` block above is the React opt-in for the ui-health
 * `WidgetDeclaration` contract (slot INLINE). No runtime consumer ships yet;
 * the block makes this component discoverable when one does.
 */
export function PostureBadge({ scenario, topN = 3 }: { scenario: string; topN?: number }) {
  const { t } = useTranslation();

  const query = useQuery({
    queryKey: ["validate-badge", scenario],
    queryFn: () => validationClient.validateScenario({ scenario }),
    enabled: scenario.length > 0,
  });

  const data = query.data;
  const passed = data?.passed ?? false;
  const top = data
    ? [...data.findings]
        .sort((a, b) => compareSeverity(a.severity, b.severity) || a.ruleId.localeCompare(b.ruleId))
        .filter((f) => f.severity === Severity.ERROR || f.severity === Severity.WARNING)
        .slice(0, topN)
    : [];

  return (
    <div
      data-testid={selectors.widget.badge}
      className="inline-flex w-full max-w-sm flex-col gap-2 rounded-xl border border-app-border bg-app-surface p-3"
    >
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.widget.title)}
        </span>
        <span className="ms-auto font-mono text-xs text-app-muted-foreground">{scenario}</span>
      </div>

      {query.isLoading && (
        <p data-testid={selectors.widget.loading} className="text-sm text-app-muted-foreground">
          {t(strings.widget.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.widget.error} className="text-sm text-red-400">
          {t(strings.widget.error)}
        </p>
      )}

      {data && (
        <>
          <div className="flex items-center gap-2">
            <span
              data-testid={selectors.widget.status}
              data-passed={passed}
              className={[
                "inline-flex items-center gap-1.5 rounded-control px-2 py-0.5 text-sm font-medium",
                passed
                  ? "border border-emerald-500/40 bg-emerald-500/10 text-emerald-300"
                  : "border border-red-500/40 bg-red-500/10 text-red-300",
              ].join(" ")}
            >
              {passed ? (
                <ShieldCheck aria-hidden="true" className="h-4 w-4" />
              ) : (
                <ShieldAlert aria-hidden="true" className="h-4 w-4" />
              )}
              {passed ? t(strings.widget.clean) : t(strings.posture.failed)}
            </span>
            <span data-testid={selectors.widget.counts} className="text-xs text-app-muted-foreground">
              {t(strings.posture.summary, {
                errors: data.summary?.errors ?? 0,
                warnings: data.summary?.warnings ?? 0,
                infos: data.summary?.infos ?? 0,
              })}
            </span>
          </div>

          {top.length > 0 && (
            <ul data-testid={selectors.widget.topFindings} className="flex flex-col gap-1">
              {top.map((f, i) => {
                const meta = severityMeta(f.severity);
                return (
                  <li key={`${f.ruleId}:${i}`} className="flex items-center gap-1.5 text-xs">
                    <span className={["rounded px-1 py-0.5 font-semibold uppercase", meta.chipClass].join(" ")}>
                      {t(meta.labelKey)}
                    </span>
                    <span className="truncate">{f.title}</span>
                  </li>
                );
              })}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
