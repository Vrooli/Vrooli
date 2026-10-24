import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { ShieldCheck, ShieldAlert } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { findingsFromResponse, passedFromResponse, summaryFromResponse, validationClient } from "../../api/validation";
import { FindingList } from "./FindingList";

const DEFAULT_SCENARIO = "security-health";

/**
 * PostureCard is the headline validation surface: enter a scenario id, run the
 * applicable security scanners through `ScenarioValidationService`, and render
 * the normalized assessment findings severity-grouped with remediation. The
 * pass/fail verdict mirrors the shared validation status.
 *
 * `scanner` and `severity` filters (e.g. the Secrets page) are applied by the
 * caller via the optional `scannerFilter` prop; the data fetch is shared.
 */
export function PostureCard({ scannerFilter }: { scannerFilter?: string } = {}) {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState(DEFAULT_SCENARIO);
  const [submitted, setSubmitted] = useState(DEFAULT_SCENARIO);

  const query = useQuery({
    queryKey: ["validate", submitted],
    queryFn: () => validationClient.validateScenario({ scenario: submitted }),
    enabled: submitted.length > 0,
  });

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(scenario.trim());
  };

  const data = query.data;
  const allFindings = findingsFromResponse(data);
  const findings = data
    ? scannerFilter
      ? allFindings.filter((f) => f.scanner === scannerFilter)
      : allFindings
    : [];
  const passed = passedFromResponse(data);
  const summary = summaryFromResponse(data);

  return (
    <section
      data-testid={selectors.posture.card}
      aria-label={t(strings.posture.title)}
      className="rounded-xl border border-app-border bg-app-surface p-4"
    >
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-app-muted-foreground">{t(strings.posture.scenarioLabel)}</span>
          <Input
            data-testid={selectors.posture.scenarioInput}
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            placeholder={t(strings.posture.scenarioPlaceholder)}
            aria-label={t(strings.posture.scenarioLabel)}
          />
        </label>
        <Button data-testid={selectors.posture.scanButton} type="submit" disabled={query.isFetching}>
          {query.isFetching ? t(strings.posture.scanning) : t(strings.posture.scan)}
        </Button>
      </form>

      {query.isLoading && (
        <p data-testid={selectors.posture.loading} className="mt-4 text-app-muted-foreground">
          {t(strings.posture.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.posture.error} className="mt-4 text-red-400">
          {errorMessage(query.error, t)}
        </p>
      )}

      {data && (
        <div className="mt-4 flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-3">
            <span
              data-testid={selectors.posture.status}
              data-passed={passed}
              className={[
                "inline-flex items-center gap-1.5 rounded-control px-2.5 py-1 text-sm font-medium",
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
              {passed ? t(strings.posture.passed) : t(strings.posture.failed)}
            </span>
            <span data-testid={selectors.posture.summary} className="text-sm text-app-muted-foreground">
              {t(strings.posture.summary, {
                errors: summary.errors,
                warnings: summary.warnings,
                infos: summary.infos,
              })}
            </span>
            {query.dataUpdatedAt > 0 && (
              <span className="text-xs text-app-muted-foreground">
                {t(strings.posture.lastScan)}{" "}
                {formatDate(new Date(query.dataUpdatedAt), { dateStyle: "medium", timeStyle: "short" })}
              </span>
            )}
          </div>

          {findings.length === 0 ? (
            <p data-testid={selectors.posture.empty} className="text-emerald-300">
              {t(strings.posture.empty)}
            </p>
          ) : (
            <FindingList findings={findings} />
          )}
        </div>
      )}
    </section>
  );
}
