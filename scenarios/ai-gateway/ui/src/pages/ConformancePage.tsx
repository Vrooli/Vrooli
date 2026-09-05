import { useMutation } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useState } from "react";

import { scanScenario } from "../api/gateway";
import { StatusChip } from "../components/StatusChip";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const severityTone = (severity: string) => {
  const normalized = severity.toLowerCase();
  if (normalized.includes("critical") || normalized.includes("high")) return "danger";
  if (normalized.includes("medium")) return "warning";
  if (normalized.includes("low")) return "info";
  return "neutral";
};

export function ConformancePage() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("ai-gateway");
  const scanMutation = useMutation({
    mutationFn: () => scanScenario(scenario),
  });

  return (
    <section
      data-testid={selectors.pages.conformance}
      aria-labelledby="conformance-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-2">
        <p className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.conformance.eyebrow)}
        </p>
        <h2 id="conformance-heading" className="text-2xl font-semibold">
          {t(strings.pages.conformance.title)}
        </h2>
        <p className="max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.conformance.description)}
        </p>
      </header>

      <form
        data-testid={selectors.conformance.form}
        aria-label={t(strings.pages.conformance.formLabel)}
        className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 md:flex-row md:items-end"
        onSubmit={(event) => {
          event.preventDefault();
          scanMutation.mutate();
        }}
      >
        <label className="grid flex-1 gap-1 text-sm font-medium">
          {t(strings.pages.conformance.scenario)}
          <input
            data-testid={selectors.conformance.scenarioInput}
            value={scenario}
            onChange={(event) => setScenario(event.target.value)}
            className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 font-mono text-sm"
          />
        </label>
        <button
          type="submit"
          data-testid={selectors.conformance.submit}
          className="inline-flex min-h-10 items-center justify-center gap-2 rounded-control bg-app-primary px-4 text-sm font-semibold text-app-primary-foreground"
        >
          <Search aria-hidden="true" size={16} />
          {scanMutation.isPending ? t(strings.states.loading) : t(strings.pages.conformance.scan)}
        </button>
      </form>

      {scanMutation.isError ? (
        <div data-testid={selectors.conformance.error} className="rounded-panel border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {errorMessage(scanMutation.error, t)}
        </div>
      ) : null}

      <div
        data-testid={selectors.conformance.result}
        role="region"
        aria-label={t(strings.pages.conformance.resultLabel)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        {scanMutation.data ? (
          <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="font-semibold">{scanMutation.data.scenario}</h3>
                <p className="mt-1 text-sm text-app-muted-foreground">
                  {t(strings.pages.conformance.maturity)}: {scanMutation.data.maturityLevel || t(strings.states.unknown)}
                </p>
              </div>
              <StatusChip tone={scanMutation.data.findings.length === 0 ? "success" : "warning"}>
                {t(strings.pages.conformance.findingCount, { count: scanMutation.data.findings.length })}
              </StatusChip>
            </div>

            {scanMutation.data.findings.length === 0 ? (
              <p className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm text-app-muted-foreground">
                {t(strings.pages.conformance.empty)}
              </p>
            ) : (
              <div className="overflow-hidden rounded-panel border border-app-border">
                <table className="w-full min-w-[760px] text-left text-sm">
                  <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                    <tr>
                      <th className="px-4 py-3">{t(strings.pages.conformance.columns.rule)}</th>
                      <th className="px-4 py-3">{t(strings.pages.conformance.columns.severity)}</th>
                      <th className="px-4 py-3">{t(strings.pages.conformance.columns.path)}</th>
                      <th className="px-4 py-3">{t(strings.pages.conformance.columns.message)}</th>
                      <th className="px-4 py-3">{t(strings.pages.conformance.columns.remediation)}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-app-border">
                    {scanMutation.data.findings.map((finding) => (
                      <tr key={`${finding.ruleId}-${finding.path}-${finding.message}`}>
                        <td className="px-4 py-3 font-mono text-xs">{finding.ruleId}</td>
                        <td className="px-4 py-3">
                          <StatusChip tone={severityTone(finding.severity)}>{finding.severity}</StatusChip>
                        </td>
                        <td className="px-4 py-3 font-mono text-xs">{finding.path}</td>
                        <td className="px-4 py-3">{finding.message}</td>
                        <td className="px-4 py-3 text-app-muted-foreground">{finding.remediation}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {scanMutation.data.recommendations.length > 0 ? (
              <div>
                <h4 className="text-sm font-semibold">{t(strings.pages.conformance.recommendations)}</h4>
                <ul className="mt-2 grid gap-2 text-sm text-app-muted-foreground">
                  {scanMutation.data.recommendations.map((recommendation) => (
                    <li key={recommendation} className="rounded-control border border-app-border bg-app-surface-muted p-3">
                      {recommendation}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </div>
        ) : (
          <p className="text-sm text-app-muted-foreground">{t(strings.pages.conformance.initial)}</p>
        )}
      </div>
    </section>
  );
}
