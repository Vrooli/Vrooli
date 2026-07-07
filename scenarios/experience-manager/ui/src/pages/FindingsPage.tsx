import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { AlertTriangle, FileSearch, Gauge } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import type { FixResponse } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import {
  applyFindingsFixes,
  fetchFindings,
  previewFindingsFixes,
} from "../api/experience";
import { PageFrame } from "../components/PageFrame";
import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fixPreviewText, uniqueRuleIDs } from "./experiencePageUtils";

export function FindingsPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const [preview, setPreview] = useState<FixResponse>();
  const [fixError, setFixError] = useState("");
  const [isFixing, setIsFixing] = useState(false);
  const {
    data: findings,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-findings", scenario],
    queryFn: () => fetchFindings(scenario),
    staleTime: 60_000,
  });
  const rows = findings ?? [];
  const previewFixes = async () => {
    setIsFixing(true);
    setFixError("");
    try {
      setPreview(await previewFindingsFixes(scenario));
    } catch (err) {
      setFixError(err instanceof Error ? err.message : t(strings.experience.findings.previewError));
    } finally {
      setIsFixing(false);
    }
  };
  const applyFixes = async () => {
    setIsFixing(true);
    setFixError("");
    try {
      const result = await applyFindingsFixes(scenario, uniqueRuleIDs(preview));
      setPreview(result.preview);
      await refetch();
    } catch (err) {
      setFixError(err instanceof Error ? err.message : t(strings.experience.findings.applyError));
    } finally {
      setIsFixing(false);
    }
  };

  return (
    <PageFrame
      testId={selectors.pages.findings}
      title={t(strings.experience.findings.title)}
      description={t(strings.experience.findings.description)}
    >
      <ul
        data-testid={selectors.experience.findings.findingsList}
        aria-label={t(strings.experience.findings.listLabel)}
        className="grid gap-3"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.loadingFindings)}
          </li>
        ) : isError ? (
          <li role="alert" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.loadError)}
          </li>
        ) : rows.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.emptyFindings)}
          </li>
        ) : (
          rows.map((finding) => (
          <li key={`${finding.code}:${finding.location}:${finding.message}`} className="rounded-panel border border-app-border bg-app-surface p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <span
                  data-testid={selectors.experience.findings.severityLabel}
                  role="note"
                  className="inline-flex items-center gap-2 text-xs font-semibold uppercase text-app-warning"
                >
                  <AlertTriangle className="size-4" aria-hidden="true" />
                  {finding.severity}
                </span>
                <p className="mt-2 font-medium">{finding.code}</p>
                <p className="mt-1 text-sm text-app-muted-foreground">{finding.message || finding.remediation}</p>
                <Link
                  data-testid={selectors.experience.findings.evidenceLink}
                  to={`/scenarios/${scenario}/pages/findings/evidence`}
                  aria-label={`${t(strings.experience.common.viewEvidence)} ${finding.code}`}
                  className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
                >
                  {t(strings.experience.common.viewEvidence)}
                </Link>
              </div>
              <div className="flex gap-2">
                <Button
                  data-testid={selectors.experience.findings.previewAction}
                  type="button"
                  variant="outline"
                  onClick={() => void previewFixes()}
                  disabled={isFixing || isFetching}
                >
                  <FileSearch className="mr-2 size-4" aria-hidden="true" />
                  {isFixing ? t(strings.experience.explorer.refreshing) : t(strings.experience.findings.preview)}
                </Button>
                <Button
                  data-testid={selectors.experience.findings.applyAction}
                  type="button"
                  onClick={() => void applyFixes()}
                  disabled={isFixing || uniqueRuleIDs(preview).length === 0}
                >
                  <Gauge className="mr-2 size-4" aria-hidden="true" />
                  {t(strings.experience.findings.apply)}
                </Button>
              </div>
            </div>
          </li>
          ))
        )}
      </ul>
      <section
        role={fixError ? "alert" : "status"}
        aria-label={t(strings.experience.findings.fixPreviewLabel)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="font-semibold">{t(strings.experience.findings.fixPreviewLabel)}</h3>
        <pre className="mt-3 overflow-auto whitespace-pre-wrap rounded-control bg-app-surface-muted p-3 text-xs">
          {fixError || fixPreviewText(preview, t(strings.experience.findings.previewEmpty))}
        </pre>
      </section>
    </PageFrame>
  );
}
