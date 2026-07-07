import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useParams } from "react-router-dom";

import { fetchEvidence, recaptureScenario } from "../api/experience";
import { PageFrame } from "../components/PageFrame";
import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import {
  captureImageSource,
  evidenceIsStale,
  formatAXNode,
  formatEvidenceMeta,
  newestEvidence,
} from "./experiencePageUtils";

export function EvidencePage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const page = params.page ?? "fleet";
  const [selectedID, setSelectedID] = useState("");
  const {
    data: evidence,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-evidence", scenario, page],
    queryFn: () => fetchEvidence({ scenario, page }),
    staleTime: 60_000,
  });
  const rows = useMemo(() => evidence ?? [], [evidence]);
  const selected = useMemo(
    () => rows.find((row) => row.id === selectedID) ?? newestEvidence(rows),
    [rows, selectedID],
  );
  const captureRef = selected?.captureRef ?? "";
  const captureSrc = captureImageSource(captureRef);
  const axNode = formatAXNode(selected?.axNodeJson ?? "");
  const stale = evidenceIsStale(selected);
  const recapture = async () => {
    await recaptureScenario(scenario);
    await refetch();
  };

  return (
    <PageFrame
      testId={selectors.pages.evidence}
      title={t(strings.experience.evidence.title)}
      description={t(strings.experience.evidence.description)}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <div className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4">
          {isLoading ? (
            <div
              data-testid={selectors.experience.evidence.captureImage}
              role="img"
              aria-label={t(strings.experience.evidence.captureLabel)}
              className="flex min-h-72 w-full items-center justify-center rounded-control border border-dashed border-app-border bg-app-surface-muted text-sm text-app-muted-foreground"
            >
              {t(strings.experience.evidence.loadingEvidence)}
            </div>
          ) : captureSrc ? (
            <img
              data-testid={selectors.experience.evidence.captureImage}
              src={captureSrc}
              alt={t(strings.experience.evidence.captureLabel)}
              className="min-h-72 w-full rounded-control border border-dashed border-app-border bg-app-surface-muted object-cover"
            />
          ) : (
            <div
              data-testid={selectors.experience.evidence.captureImage}
              role="img"
              aria-label={t(strings.experience.evidence.captureLabel)}
              className="flex min-h-72 w-full flex-col items-center justify-center rounded-control border border-dashed border-app-border bg-app-surface-muted p-4 text-center text-sm text-app-muted-foreground"
            >
              <span>{rows.length === 0 ? t(strings.experience.evidence.emptyEvidence) : t(strings.experience.evidence.captureReference)}</span>
              {captureRef ? <code className="mt-2 max-w-full break-all text-xs">{captureRef}</code> : null}
            </div>
          )}
          <Button
            data-testid={selectors.experience.evidence.recaptureAction}
            type="button"
            className="mt-4"
            onClick={() => void recapture()}
          >
            <RefreshCw className="mr-2 size-4" aria-hidden="true" />
            {isFetching ? t(strings.experience.evidence.refreshing) : t(strings.experience.evidence.recapture)}
          </Button>
          {stale ? <p className="mt-2 text-sm text-app-warning">{t(strings.experience.evidence.staleEvidence)}</p> : null}
        </div>
        <div
          data-testid={selectors.experience.evidence.treePanel}
          role={isError ? "alert" : "region"}
          aria-label={t(strings.experience.evidence.treeLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.evidence.treeLabel)}</h3>
          <pre className="mt-3 max-w-full overflow-auto rounded-control bg-app-surface-muted p-3 text-xs">
            {isError
              ? t(strings.experience.evidence.loadError)
              : axNode || t(strings.experience.evidence.emptyTree)}
          </pre>
        </div>
      </div>
      <ul
        data-testid={selectors.experience.evidence.verdictList}
        aria-label={t(strings.experience.evidence.verdictsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.evidence.loadingEvidence)}
          </li>
        ) : rows.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.evidence.emptyVerdicts)}
          </li>
        ) : (
          rows.map((row) => (
            <li key={row.id} className="rounded-panel border border-app-border bg-app-surface p-4">
              <p className="font-medium">{row.claim}</p>
              <p className="text-sm text-app-muted-foreground">{formatEvidenceMeta(row)}</p>
              {row.message ? <p className="mt-2 text-sm text-app-muted-foreground">{row.message}</p> : null}
              <button
                data-testid={selectors.experience.evidence.evidenceLink}
                type="button"
                onClick={() => setSelectedID(row.id)}
                aria-label={`${t(strings.experience.common.viewEvidence)} ${row.claim}`}
                className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
              >
                {t(strings.experience.common.viewEvidence)}
              </button>
            </li>
          ))
        )}
      </ul>
    </PageFrame>
  );
}
