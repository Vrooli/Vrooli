import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { JsonObject } from "@bufbuild/protobuf";

import { researchClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const INVESTIGATION_KEY = "web-search.active-investigation";

function firstPassageId(result: JsonObject | undefined): string {
  const assessments = result?.["assessments"];
  if (!Array.isArray(assessments)) return "";
  for (const assessment of assessments) {
    if (typeof assessment !== "object" || assessment === null || Array.isArray(assessment)) continue;
    const evidence = assessment["evidence"];
    if (!Array.isArray(evidence)) continue;
    for (const reference of evidence) {
      if (typeof reference !== "object" || reference === null || Array.isArray(reference)) continue;
      const passageId = reference["passage_id"];
      if (typeof passageId === "string" && passageId.trim()) return passageId;
    }
  }
  return "";
}

function resultStatus(result: JsonObject | undefined): string {
  const value = result?.["status"];
  return typeof value === "string" ? value : "";
}

function resultGaps(result: JsonObject | undefined): string[] {
  const value = result?.["gaps"];
  if (!Array.isArray(value)) return [];
  return value.filter((gap): gap is string => typeof gap === "string" && gap.trim().length > 0);
}

/**
 * Investigation controls use the declared run handle as the durable UI
 * identity. A timeout leaves that handle in local storage, so reload and
 * resume attach to the existing Agent Manager execution.
 */
export function ResearchPanel() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [runId, setRunId] = useState(() => window.localStorage.getItem(INVESTIGATION_KEY) ?? "");
  const [passageId, setPassageId] = useState("");

  const wait = useMutation({
    mutationFn: (id: string) => researchClient.waitResearch({ runId: id, timeoutSeconds: 30 }),
  });
  const start = useMutation({
    mutationFn: () =>
      researchClient.runL3({
        query: query.trim(),
        idempotencyKey: `ui-${crypto.randomUUID()}`,
        policy: { topN: 5, minimumSources: 1, maxEvidenceBytes: 120000 },
      }),
    onSuccess: (response) => {
      setRunId(response.runId);
      window.localStorage.setItem(INVESTIGATION_KEY, response.runId);
      wait.mutate(response.runId);
    },
  });
  const passage = useMutation({
    mutationFn: (id: string) => researchClient.getEvidencePassage({ passageId: id.trim() }),
  });

  useEffect(() => {
    if (!runId) return;
    window.localStorage.setItem(INVESTIGATION_KEY, runId);
  }, [runId]);

  const error = start.error ?? wait.error ?? passage.error;
  const status = wait.data;
  const resolvedResultStatus = resultStatus(status?.result);
  const gaps = resultGaps(status?.result);
  const assessmentPassageId = firstPassageId(status?.result);
  useEffect(() => {
    if (!assessmentPassageId || passageId || passage.data || passage.isPending) return;
    setPassageId(assessmentPassageId);
    passage.mutate(assessmentPassageId);
  }, [assessmentPassageId, passage, passageId]);
  return (
    <section
      data-testid={selectors.research.panel}
      aria-labelledby="research-investigation-heading"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="research-investigation-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.research.heading)}
      </h3>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          data-testid={selectors.research.query}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t(strings.research.queryPlaceholder)}
          aria-label={t(strings.research.queryPlaceholder)}
        />
        <Button data-testid={selectors.research.start} disabled={!query.trim() || start.isPending} onClick={() => start.mutate()}>
          {t(strings.research.start)}
        </Button>
        {runId && (
          <Button data-testid={selectors.research.resume} variant="outline" disabled={wait.isPending} onClick={() => wait.mutate(runId)}>
            {t(strings.research.resume)}
          </Button>
        )}
      </div>
      {wait.isPending && <p className="text-sm text-app-muted-foreground">{t(strings.research.waiting)}</p>}
      {status && (
        <div data-testid={selectors.research.status} className="flex flex-col gap-2 text-sm">
          <p>{t(strings.research.status, { status: status.status })}</p>
          {resolvedResultStatus && resolvedResultStatus !== status.status && (
            <p data-testid={selectors.research.resultStatus} data-result-status={resolvedResultStatus}>{t(strings.research.status, { status: resolvedResultStatus })}</p>
          )}
          {status.summary && <p data-testid={selectors.research.summary}>{status.summary}</p>}
          {gaps.length > 0 && (
            <ul data-testid={selectors.research.gaps} aria-label={t(strings.research.summary)}>
              {gaps.map((gap) => <li key={gap}>{gap}</li>)}
            </ul>
          )}
        </div>
      )}
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          data-testid={selectors.research.passageId}
          value={passageId}
          onChange={(e) => setPassageId(e.target.value)}
          placeholder={t(strings.research.passagePlaceholder)}
          aria-label={t(strings.research.passagePlaceholder)}
        />
        <Button data-testid={selectors.research.readPassage} variant="outline" disabled={!passageId.trim() || passage.isPending} onClick={() => passage.mutate(passageId.trim())}>
          {t(strings.research.readPassage)}
        </Button>
      </div>
      {passage.data && <blockquote data-testid={selectors.research.passage} className="border-s-2 border-app-primary ps-3 text-sm">{passage.data.content}</blockquote>}
      {error && <p data-testid={selectors.research.error} className="text-sm text-app-danger">{t(strings.research.error, { message: errorMessage(error, t) })}</p>}
    </section>
  );
}
