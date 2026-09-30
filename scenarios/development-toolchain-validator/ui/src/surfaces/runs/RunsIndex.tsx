import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { PlayCircle } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  validationRunClient,
  type ValidationRun,
} from "../../api/validationRun";
import { TupleKind } from "../../api/validationRecord";
import { errorMessage } from "../../lib/errorMessage";
import { ROUTES } from "../../routes.generated";
import { runStatusLabelKey } from "../../lib/runStatus";
import { Button } from "../../shared/ui/primitives/Button";
import { Input } from "../../shared/ui/primitives/Input";
import { Card } from "../../shared/ui/primitives/Card";
import { Badge } from "../../shared/ui/primitives/Badge";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";

const ACTIVE_RUNS_QUERY_KEY = ["activeRuns"] as const;

interface StartFormState {
  kind: "skill" | "tool";
  subjectId: string;
  goldenSlug: string;
  force: boolean;
}

const EMPTY_FORM: StartFormState = {
  kind: "skill",
  subjectId: "",
  goldenSlug: "",
  force: false,
};

/**
 * Runs index surface — the operator's launchpad for validation runs.
 *
 * Lists the non-terminal runs returned by ListActive (polled while any
 * are in flight) and exposes a start form that maps directly onto the
 * ValidationRunService.Start RPC. Terminal history lives on the goldens /
 * tuple-detail surfaces, which read from the report + record domains.
 */
export function RunsIndex() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [form, setForm] = useState<StartFormState>(EMPTY_FORM);

  const activeQuery = useQuery({
    queryKey: ACTIVE_RUNS_QUERY_KEY,
    queryFn: () => validationRunClient.listActive({}),
    // Poll while runs are in flight so the list drains as they finish.
    refetchInterval: (query) =>
      (query.state.data?.runs.length ?? 0) > 0 ? 2_000 : false,
  });

  const startMutation = useMutation({
    mutationFn: (input: StartFormState) =>
      validationRunClient.start({
        tupleKind: input.kind === "tool" ? TupleKind.TOOL : TupleKind.SKILL,
        subjectId: input.subjectId,
        goldenSlug: input.goldenSlug,
        force: input.force,
      }),
    onSuccess: (resp) => {
      setForm(EMPTY_FORM);
      void queryClient.invalidateQueries({ queryKey: ACTIVE_RUNS_QUERY_KEY });
      if (resp.run?.id) {
        void navigate(ROUTES.runDetail(resp.run.id));
      }
    },
  });

  const handleStart = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    startMutation.mutate(form);
  };

  const runs = activeQuery.data?.runs ?? [];

  return (
    <section
      data-testid={selectors.runs.surface}
      aria-labelledby={selectors.runs.surface}
      className="flex flex-col gap-4"
    >
      <PanelHeader
        title={t(strings.runs.title)}
        description={t(strings.runs.subtitle)}
      />

      <Card surface="raised" data-testid={selectors.runs.startCard}>
        <h2 className="mb-3 text-sm font-semibold text-app-foreground">
          {t(strings.runs.startTitle)}
        </h2>
        <form
          data-testid={selectors.runs.startForm}
          className="grid gap-3"
          onSubmit={handleStart}
        >
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.runs.kindLabel)}
            <select
              data-testid={selectors.runs.startKind}
              value={form.kind}
              onChange={(e) =>
                setForm({ ...form, kind: e.target.value as "skill" | "tool" })
              }
              className="mt-1 w-full rounded-control border border-app-border bg-app-surface px-2 py-1.5 text-sm text-app-foreground"
            >
              <option value="skill">{t(strings.runs.kindSkill)}</option>
              <option value="tool">{t(strings.runs.kindTool)}</option>
            </select>
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.runs.subjectLabel)}
            <Input
              data-testid={selectors.runs.startSubject}
              value={form.subjectId}
              onChange={(e) => setForm({ ...form, subjectId: e.target.value })}
              required
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.runs.goldenLabel)}
            <Input
              data-testid={selectors.runs.startGolden}
              value={form.goldenSlug}
              onChange={(e) => setForm({ ...form, goldenSlug: e.target.value })}
              required
            />
          </label>
          <label className="flex items-center gap-2 text-xs text-app-muted-foreground">
            <input
              type="checkbox"
              data-testid={selectors.runs.startForce}
              checked={form.force}
              onChange={(e) => setForm({ ...form, force: e.target.checked })}
            />
            {t(strings.runs.forceLabel)}
          </label>
          <Button
            data-testid={selectors.runs.startSubmit}
            type="submit"
            disabled={startMutation.isPending}
          >
            {startMutation.isPending
              ? t(strings.runs.starting)
              : t(strings.runs.startSubmit)}
          </Button>
          {startMutation.error ? (
            <p
              data-testid={selectors.runs.startError}
              className="text-sm text-status-failure"
            >
              {errorMessage(startMutation.error, t)}
            </p>
          ) : null}
        </form>
      </Card>

      <h2 className="text-sm font-semibold text-app-foreground">
        {t(strings.runs.activeTitle)}
      </h2>

      {activeQuery.isLoading ? (
        <LoadingSkeleton
          data-testid={selectors.runs.loading}
          variant="card"
          count={2}
        />
      ) : null}

      {activeQuery.error ? (
        <p
          data-testid={selectors.runs.error}
          className="text-sm text-status-failure"
        >
          {errorMessage(activeQuery.error, t)}
        </p>
      ) : null}

      {!activeQuery.isLoading && runs.length === 0 && !activeQuery.error ? (
        <EmptyState
          testId={selectors.runs.empty}
          icon={<PlayCircle className="h-8 w-8" aria-hidden />}
          title={t(strings.runs.empty)}
          description={t(strings.runs.emptyDescription)}
        />
      ) : null}

      {runs.length > 0 ? (
        <ul data-testid={selectors.runs.list} className="flex flex-col gap-2">
          {runs.map((run: ValidationRun) => (
            <li key={run.id}>
              <button
                type="button"
                data-testid={selectors.runs.row}
                onClick={() => void navigate(ROUTES.runDetail(run.id))}
                className="w-full text-left"
              >
                <Card
                  surface="raised"
                  className="transition-colors hover:border-app-accent"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-app-foreground">
                        {t(strings.runs.rowLabel, {
                          subject: run.subjectId,
                          golden: run.goldenSlug,
                        })}
                      </p>
                      <p className="mt-1 truncate font-mono text-[11px] text-app-muted-foreground">
                        {run.id}
                      </p>
                    </div>
                    <Badge variant="neutral">
                      {t(runStatusLabelKey(run.status))}
                    </Badge>
                  </div>
                </Card>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
