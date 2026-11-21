import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { ShieldCheck, ShieldX } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { BudgetSchema } from "@vrooli/proto-types/performance-health/v1/budgets/budgets_pb";
import { perfClient, type Budget } from "../../api/perf";
import { useScenario } from "../perf/scenarioContextValue";
import { ScenarioPicker } from "../perf/ScenarioPicker";
import { formatMs } from "../perf/format";

/** Editable budget fields, mapped to proto field names. */
type IntField =
  | "goBuildMaxMs"
  | "uiBuildMaxMs"
  | "bundleMaxBytes"
  | "lcpMaxMs"
  | "startupMaxMs";

type LeafKeys<T> = T extends string ? T : T extends object ? LeafKeys<T[keyof T]> : never;
type TKey = LeafKeys<typeof strings>;

const INT_FIELDS: { key: IntField; labelKey: TKey }[] = [
  { key: "goBuildMaxMs", labelKey: strings.budgets.field.goBuild },
  { key: "uiBuildMaxMs", labelKey: strings.budgets.field.uiBuild },
  { key: "bundleMaxBytes", labelKey: strings.budgets.field.bundle },
  { key: "lcpMaxMs", labelKey: strings.budgets.field.lcp },
  { key: "startupMaxMs", labelKey: strings.budgets.field.startup },
];

/**
 * "Budgets" workflow. GetBudget loads the per-scenario thresholds (and whether a
 * budget is declared); the form edits them and SetBudget persists (with a
 * dry-run preview). CheckBudget runs the gate and lists any violations. All real
 * data — edits round-trip through the budget store.
 */
export function BudgetsPanel() {
  const { t } = useTranslation();
  const { scenario } = useScenario();
  const queryClient = useQueryClient();

  const budgetQuery = useQuery({
    queryKey: ["budget", scenario],
    queryFn: () => perfClient.getBudget({ scenario }),
  });

  const [draft, setDraft] = useState<Record<IntField, string>>(() => emptyDraft());
  const [ratchet, setRatchet] = useState(false);

  // Reseed the form whenever the loaded budget changes.
  useEffect(() => {
    const b = budgetQuery.data?.budget;
    if (!b) return;
    setDraft({
      goBuildMaxMs: intToInput(b.goBuildMaxMs),
      uiBuildMaxMs: intToInput(b.uiBuildMaxMs),
      bundleMaxBytes: intToInput(b.bundleMaxBytes),
      lcpMaxMs: intToInput(b.lcpMaxMs),
      startupMaxMs: intToInput(b.startupMaxMs),
    });
    setRatchet(b.ratchet);
  }, [budgetQuery.data]);

  const save = useMutation({
    mutationFn: () =>
      perfClient.setBudget({ budget: buildBudget(scenario, draft, ratchet) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["budget", scenario] });
    },
  });

  const check = useMutation({
    mutationFn: () => perfClient.checkBudget({ scenario }),
  });

  return (
    <section
      data-testid={selectors.pages.budgets}
      aria-labelledby="budgets-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-3">
        <h2 id="budgets-heading" className="text-2xl font-semibold">
          {t(strings.budgets.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.budgets.description)}</p>
        <ScenarioPicker />
      </header>

      {budgetQuery.isLoading && (
        <LoadingState
          title={t(strings.budgets.loadingTitle)}
          skeleton={
            <div className="grid gap-4 rounded-panel border border-app-border bg-app-surface p-4 sm:grid-cols-2">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-14 w-full" />
              ))}
            </div>
          }
        />
      )}
      {budgetQuery.error && (
        <ErrorState
          testId={selectors.budgets.error}
          title={t(strings.budgets.errorTitle)}
          message={errorMessage(budgetQuery.error, t)}
          onRetry={() => void budgetQuery.refetch()}
          retrying={budgetQuery.isFetching}
        />
      )}

      {budgetQuery.data && (
        <section
          data-testid={selectors.budgets.form}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          {!budgetQuery.data.declared && (
            <p
              data-testid={selectors.budgets.notDeclared}
              className="mb-3 rounded-control border border-app-warning/40 bg-app-warning/10 px-3 py-2 text-sm text-app-warning"
            >
              {t(strings.budgets.notDeclared)}
            </p>
          )}
          <div className="grid gap-4 sm:grid-cols-2">
            {INT_FIELDS.map((f) => (
              <label key={f.key} className="flex flex-col gap-1 text-sm">
                <span className="font-medium text-app-muted-foreground">{t(f.labelKey)}</span>
                <Input
                  data-testid={selectors.budgets.field({ field: f.key })}
                  type="number"
                  inputMode="numeric"
                  min={0}
                  value={draft[f.key]}
                  onChange={(e) =>
                    setDraft((d) => ({ ...d, [f.key]: e.target.value }))
                  }
                  placeholder={t(strings.budgets.unsetPlaceholder)}
                />
              </label>
            ))}
          </div>
          <label className="mt-4 flex items-center gap-2 text-sm">
            <input
              data-testid={selectors.budgets.ratchet}
              type="checkbox"
              checked={ratchet}
              onChange={(e) => setRatchet(e.target.checked)}
              className="h-4 w-4"
            />
            <span className="text-app-foreground">{t(strings.budgets.ratchet)}</span>
          </label>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <Button
              data-testid={selectors.budgets.saveButton}
              onClick={() => save.mutate()}
              disabled={save.isPending}
            >
              {save.isPending ? t(strings.common.loading) : t(strings.budgets.save)}
            </Button>
            <Button
              data-testid={selectors.budgets.checkButton}
              variant="outline"
              onClick={() => check.mutate()}
              disabled={check.isPending}
            >
              {check.isPending ? t(strings.common.loading) : t(strings.budgets.check)}
            </Button>
            {save.isSuccess && (
              <span data-testid={selectors.budgets.saved} className="text-sm text-app-success">
                {t(strings.budgets.saved)}
              </span>
            )}
            {save.error && (
              <span className="text-sm text-app-danger">{errorMessage(save.error, t)}</span>
            )}
          </div>
        </section>
      )}

      {check.error && (
        <p data-testid={selectors.budgets.checkError} className="text-app-danger">
          {errorMessage(check.error, t)}
        </p>
      )}
      {check.data && (
        <section
          data-testid={selectors.budgets.checkResult}
          aria-label={t(strings.budgets.checkResult.title)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <div className="flex items-center gap-2">
            {check.data.passed ? (
              <ShieldCheck aria-hidden="true" className="h-5 w-5 text-app-success" />
            ) : (
              <ShieldX aria-hidden="true" className="h-5 w-5 text-app-danger" />
            )}
            <span
              data-testid={selectors.budgets.checkVerdict}
              className={check.data.passed ? "text-app-success" : "text-app-danger"}
            >
              {check.data.passed
                ? t(strings.budgets.checkResult.passed)
                : t(strings.budgets.checkResult.failed)}
            </span>
          </div>
          {check.data.violations.length > 0 && (
            <div className="mt-3 overflow-x-auto">
              <table className="w-full border-collapse text-sm">
                <thead>
                  <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
                    <th scope="col" className="px-2 py-1 text-start font-medium">
                      {t(strings.budgets.checkResult.col.axis)}
                    </th>
                    <th scope="col" className="px-2 py-1 text-end font-medium">
                      {t(strings.budgets.checkResult.col.measured)}
                    </th>
                    <th scope="col" className="px-2 py-1 text-end font-medium">
                      {t(strings.budgets.checkResult.col.budget)}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {check.data.violations.map((v) => (
                    <tr key={v.axis} className="border-t border-app-border">
                      <td className="px-2 py-1.5 font-medium text-app-foreground">{v.axis}</td>
                      <td className="px-2 py-1.5 text-end tabular-nums text-app-danger">
                        {formatMs(v.measured)}
                      </td>
                      <td className="px-2 py-1.5 text-end tabular-nums">{formatMs(v.budget)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </section>
  );
}

function emptyDraft(): Record<IntField, string> {
  return {
    goBuildMaxMs: "",
    uiBuildMaxMs: "",
    bundleMaxBytes: "",
    lcpMaxMs: "",
    startupMaxMs: "",
  };
}

function intToInput(v: bigint): string {
  return v > 0n ? String(v) : "";
}

function inputToInt(v: string): bigint {
  const n = Number(v);
  if (!Number.isFinite(n) || n <= 0) return 0n;
  return BigInt(Math.round(n));
}

function buildBudget(
  scenario: string,
  draft: Record<IntField, string>,
  ratchet: boolean,
): Budget {
  return create(BudgetSchema, {
    scenario,
    goBuildMaxMs: inputToInt(draft.goBuildMaxMs),
    uiBuildMaxMs: inputToInt(draft.uiBuildMaxMs),
    bundleMaxBytes: inputToInt(draft.bundleMaxBytes),
    lcpMaxMs: inputToInt(draft.lcpMaxMs),
    startupMaxMs: inputToInt(draft.startupMaxMs),
    ratchet,
  });
}
