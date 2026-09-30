import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Wand2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { perfClient, type ReadinessFixResponse } from "../../api/perf";
import { useScenario } from "../perf/scenarioContextValue";
import { ScenarioPicker } from "../perf/ScenarioPicker";
import { TIER_LABEL_KEY, tierChipClass, tierKey } from "../perf/format";

/**
 * "Readiness + one-click autofix" workflow. ValidateReadiness reports the
 * scenario's capture tier and any missing perf-build infra (the four Tier-1
 * pieces) as autofixable findings. PreviewReadinessFix shows what would change;
 * ApplyReadinessFix writes the format-preserving edits. After a successful apply
 * the readiness query is invalidated so the tier/findings refresh.
 */
export function ReadinessPanel() {
  const { t } = useTranslation();
  const { scenario } = useScenario();
  const queryClient = useQueryClient();
  const [lastFix, setLastFix] = useState<ReadinessFixResponse | null>(null);

  const readiness = useQuery({
    queryKey: ["readiness", scenario],
    queryFn: () => perfClient.validateReadiness({ scenario }),
  });

  const preview = useMutation({
    mutationFn: () => perfClient.previewReadinessFix({ scenario }),
    onSuccess: (res) => setLastFix(res),
  });
  const apply = useMutation({
    mutationFn: () => perfClient.applyReadinessFix({ scenario }),
    onSuccess: async (res) => {
      setLastFix(res);
      await queryClient.invalidateQueries({ queryKey: ["readiness", scenario] });
    },
  });

  const data = readiness.data;
  const findings = data?.assessment?.findings ?? [];
  const autofixable = findings.filter((f) => f.autofixAvailable);
  const autofixableCount = data?.autofixableCount ?? autofixable.length;

  return (
    <section
      data-testid={selectors.pages.readiness}
      aria-labelledby="readiness-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-3">
        <h2 id="readiness-heading" className="text-2xl font-semibold">
          {t(strings.readiness.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.readiness.description)}</p>
        <ScenarioPicker />
      </header>

      {readiness.isLoading && (
        <LoadingState
          title={t(strings.readiness.loadingTitle)}
          skeleton={
            <div className="flex flex-col gap-4">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-40 w-full" />
            </div>
          }
        />
      )}
      {readiness.error && (
        <ErrorState
          testId={selectors.readiness.error}
          title={t(strings.readiness.errorTitle)}
          message={errorMessage(readiness.error, t)}
          onRetry={() => void readiness.refetch()}
          retrying={readiness.isFetching}
        />
      )}

      {data && (
        <section
          data-testid={selectors.readiness.summary}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span
              data-testid={selectors.readiness.tierBadge}
              className={[
                "rounded-control px-2 py-1 text-xs font-semibold uppercase",
                tierChipClass(tierKey(data.tier)),
              ].join(" ")}
            >
              {t(TIER_LABEL_KEY[tierKey(data.tier)])}
            </span>
            <span className="text-app-muted-foreground">
              {t(strings.readiness.framework)}:{" "}
              <span className="text-app-foreground">{data.uiFramework || "—"}</span>
            </span>
            <span
              data-testid={selectors.readiness.autofixableCount}
              className="text-app-muted-foreground"
            >
              {t(strings.readiness.autofixableCount, { count: autofixableCount })}
            </span>
          </div>
          {data.degradedReason && (
            <p className="mt-2 text-sm text-app-warning">⚠ {data.degradedReason}</p>
          )}
        </section>
      )}

      {/* Missing infra / autofixable gaps */}
      {data && (
        <section
          data-testid={selectors.readiness.gaps}
          aria-label={t(strings.readiness.gaps.title)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <div className="flex flex-wrap items-center gap-3">
            <h3 className="text-sm font-medium text-app-muted-foreground">
              {t(strings.readiness.gaps.title)}
            </h3>
            <div className="ms-auto flex gap-2">
              <Button
                data-testid={selectors.readiness.previewButton}
                variant="outline"
                size="sm"
                onClick={() => preview.mutate()}
                disabled={preview.isPending || autofixableCount === 0}
              >
                {preview.isPending ? t(strings.common.loading) : t(strings.readiness.preview)}
              </Button>
              <Button
                data-testid={selectors.readiness.applyButton}
                size="sm"
                onClick={() => apply.mutate()}
                disabled={apply.isPending || autofixableCount === 0}
              >
                <Wand2 aria-hidden="true" className="me-1 h-4 w-4" />
                {apply.isPending ? t(strings.readiness.applying) : t(strings.readiness.apply)}
              </Button>
            </div>
          </div>

          {findings.length === 0 ? (
            <p
              data-testid={selectors.readiness.gapsEmpty}
              className="mt-3 flex items-center gap-2 text-app-success"
            >
              <CheckCircle2 aria-hidden="true" className="h-4 w-4" />
              {t(strings.readiness.gaps.empty)}
            </p>
          ) : (
            <ul className="mt-3 flex flex-col gap-2">
              {findings.map((f) => (
                <li
                  key={`${f.code}:${f.location}`}
                  data-testid={selectors.readiness.gapRow({ code: f.code })}
                  className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium text-app-foreground">{f.title || f.code}</span>
                    {f.autofixAvailable && (
                      <span className="rounded-control border border-app-info/40 bg-app-info/10 px-1.5 py-0.5 text-xs text-app-info">
                        {t(strings.readiness.gaps.autofixable)}
                      </span>
                    )}
                  </div>
                  {f.message && <p className="text-app-muted-foreground">{f.message}</p>}
                  {f.remediation && (
                    <p className="text-app-muted-foreground">{f.remediation}</p>
                  )}
                  {f.location && (
                    <p className="font-mono text-xs text-app-muted-foreground">{f.location}</p>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Fix result */}
      {apply.error && (
        <p data-testid={selectors.readiness.applyError} className="text-app-danger">
          {errorMessage(apply.error, t)}
        </p>
      )}
      {lastFix && <FixResult result={lastFix} />}
    </section>
  );
}

function FixResult({ result }: { result: ReadinessFixResponse }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.readiness.fixResult}
      aria-label={t(strings.readiness.fixResult.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.readiness.fixResult.title)}
      </h3>
      <p className="mt-2 text-sm">
        <span
          className={[
            "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
            result.applied
              ? "border border-app-success/40 bg-app-success/10 text-app-success"
              : "border border-app-info/40 bg-app-info/10 text-app-info",
          ].join(" ")}
        >
          {result.applied
            ? t(strings.readiness.fixResult.applied)
            : t(strings.readiness.fixResult.previewed)}
        </span>
      </p>
      {result.candidates.length > 0 && (
        <ul className="mt-3 flex flex-col gap-1 text-sm">
          {result.candidates.map((c) => (
            <li key={`${c.ruleId}:${c.filePath}`} className="flex flex-col">
              <span className="font-medium text-app-foreground">{c.ruleId}</span>
              <span className="font-mono text-xs text-app-muted-foreground">{c.filePath}</span>
              {c.description && (
                <span className="text-app-muted-foreground">{c.description}</span>
              )}
            </li>
          ))}
        </ul>
      )}
      {result.messages.length > 0 && (
        <ul className="mt-3 flex flex-col gap-1 text-sm text-app-muted-foreground">
          {result.messages.map((m, i) => (
            <li key={i}>{m}</li>
          ))}
        </ul>
      )}
    </section>
  );
}
