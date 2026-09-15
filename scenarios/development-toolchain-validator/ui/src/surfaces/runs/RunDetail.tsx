import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import {
  validationRunClient,
  ValidationRunStatus,
} from "../../api/validationRun";
import { errorMessage } from "../../lib/errorMessage";
import { runStatusLabelKey, runVerdictLabelKey } from "../../lib/runStatus";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { Badge } from "../../shared/ui/primitives/Badge";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { MetadataList, type MetadataItem } from "../../shared/ui/composites/MetadataList";

const DATE_OPTS: Intl.DateTimeFormatOptions = {
  dateStyle: "medium",
  timeStyle: "medium",
};

/**
 * Run detail surface — single validation run, polled until it reaches a
 * terminal status. Shows the operational status, the terminal verdict
 * once available, run provenance (agent-manager run id, timestamps), and
 * any error message. History across runs lives on the tuple-detail view.
 */
export function RunDetail() {
  const { t } = useTranslation();
  const params = useParams<{ id: string }>();
  const id = params.id ?? "";
  const navigate = useNavigate();

  const runQuery = useQuery({
    queryKey: ["run", id] as const,
    queryFn: () => validationRunClient.get({ id }),
    enabled: id.length > 0,
    // Poll until the run is terminal, then stop.
    refetchInterval: (query) =>
      query.state.data?.run?.status === ValidationRunStatus.TERMINAL
        ? false
        : 2_000,
  });

  if (runQuery.isLoading) {
    return (
      <LoadingSkeleton
        data-testid={selectors.runs.loading}
        variant="card"
        count={2}
      />
    );
  }

  if (runQuery.error) {
    return (
      <p data-testid={selectors.runs.error} className="text-sm text-status-failure">
        {errorMessage(runQuery.error, t)}
      </p>
    );
  }

  const run = runQuery.data?.run;
  if (!run) {
    return (
      <EmptyState
        testId={selectors.runs.empty}
        title={t(strings.runs.notFound)}
        action={
          <Button
            size="sm"
            variant="outline"
            onClick={() => void navigate(ROUTES.runsIndex)}
          >
            {t(strings.runs.backToIndex)}
          </Button>
        }
      />
    );
  }

  const isTerminal = run.status === ValidationRunStatus.TERMINAL;

  const metaItems: MetadataItem[] = [
    {
      label: t(strings.runs.rowLabel, {
        subject: run.subjectId,
        golden: run.goldenSlug,
      }),
      value: run.id,
      mono: true,
    },
  ];
  if (run.agentManagerRunId) {
    metaItems.push({
      label: t(strings.runs.agentRunLabel),
      value: run.agentManagerRunId,
      mono: true,
    });
  }
  if (run.createdAt) {
    metaItems.push({
      label: t(strings.runs.createdLabel),
      value: formatDate(timestampDate(run.createdAt), DATE_OPTS),
    });
  }
  if (run.startedAt) {
    metaItems.push({
      label: t(strings.runs.startedLabel),
      value: formatDate(timestampDate(run.startedAt), DATE_OPTS),
    });
  }
  if (run.endedAt) {
    metaItems.push({
      label: t(strings.runs.endedLabel),
      value: formatDate(timestampDate(run.endedAt), DATE_OPTS),
    });
  }

  return (
    <section
      data-testid={selectors.runs.detail}
      aria-labelledby={selectors.runs.detailHeading}
      className="flex flex-col gap-5"
    >
      <PanelHeader
        title={
          <span
            data-testid={selectors.runs.detailHeading}
            id={selectors.runs.detailHeading}
          >
            {t(strings.runs.detailHeading, { id: run.id })}
          </span>
        }
        actions={
          <Button
            data-testid={selectors.runs.detailBack}
            size="sm"
            variant="ghost"
            onClick={() => void navigate(ROUTES.runsIndex)}
          >
            <ArrowLeft className="h-4 w-4" />
            {t(strings.runs.backToIndex)}
          </Button>
        }
      />

      <div className="flex flex-wrap items-center gap-3">
        <span className="text-xs text-app-muted-foreground">
          {t(strings.runs.statusLabel)}
        </span>
        <Badge data-testid={selectors.runs.detailStatus} variant="neutral">
          {t(runStatusLabelKey(run.status))}
        </Badge>
        <span className="text-xs text-app-muted-foreground">
          {t(strings.runs.verdictLabel)}
        </span>
        <Badge data-testid={selectors.runs.detailVerdict} variant="neutral">
          {isTerminal
            ? t(runVerdictLabelKey(run.terminalVerdict))
            : t(strings.runs.verdict.pending)}
        </Badge>
      </div>

      <MetadataList items={metaItems} />

      {run.errorMessage ? (
        <p
          data-testid={selectors.runs.detailError}
          className="text-sm text-status-failure"
        >
          {t(strings.runs.errorLabel)}: {run.errorMessage}
        </p>
      ) : null}
    </section>
  );
}
