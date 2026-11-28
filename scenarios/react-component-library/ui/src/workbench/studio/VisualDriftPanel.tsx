import { useEffect, useRef, useState } from "react";

import {
  acceptWorkspaceSourceBaseline,
  compareWorkspaceSourceBaseline,
  createWorkspaceSourceBaseline,
  listWorkspaceSourceBaselines,
  listWorkspaceAssets,
  listWorkspaceCaptureReceipts,
  listWorkspaceSourceBaselineComparisons,
  workspaceCaptureScreenshotUrl,
  workspaceAssetUrl,
  workspaceSourceVisualHeatmapUrl,
  type WorkspaceCaptureReceipt,
  type WorkspaceSourceBaselineSet,
  type WorkspaceSourceVisualComparison,
  type WorkspaceAsset,
} from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { DescriptionList } from "@vrooli/react-component-library/DescriptionList/1.0.7";
import "./visual-drift.css";

export function VisualDriftPanel({
  workspaceId,
  stateId,
}: {
  workspaceId: string;
  stateId?: string;
}) {
  const { t } = useTranslation();
  const requestId = useRef(0);
  const [receipts, setReceipts] = useState<WorkspaceCaptureReceipt[]>([]);
  const [sourceComparisons, setSourceComparisons] = useState<WorkspaceSourceVisualComparison[]>([]);
  const [sourceBaselines, setSourceBaselines] = useState<WorkspaceSourceBaselineSet>({
    candidates: [],
    accepted: null,
  });
  const [assets, setAssets] = useState<WorkspaceAsset[]>([]);
  const [candidateAssetId, setCandidateAssetId] = useState("");
  const [acceptanceLabel, setAcceptanceLabel] = useState("");
  const [currentId, setCurrentId] = useState("");
  const [activeComparisonId, setActiveComparisonId] = useState("");
  const [loading, setLoading] = useState(false);
  const [comparing, setComparing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const sequence = ++requestId.current;
    setComparing(false);
    if (!stateId) {
      setReceipts([]);
      setSourceBaselines({ candidates: [], accepted: null });
      setSourceComparisons([]);
      setAssets([]);
      setCandidateAssetId("");
      setCurrentId("");
      setActiveComparisonId("");
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    void Promise.all([
      listWorkspaceCaptureReceipts(workspaceId, stateId),
      listWorkspaceSourceBaselines(workspaceId, stateId),
      listWorkspaceAssets(workspaceId),
      listWorkspaceSourceBaselineComparisons(workspaceId, stateId),
    ])
      .then(([nextReceipts, nextBaselines, nextAssets, nextSourceComparisons]) => {
        if (requestId.current !== sequence) return;
        setReceipts(nextReceipts);
        setSourceBaselines(nextBaselines);
        const referenceAssets = nextAssets.filter((asset) => asset.fidelity === "reference");
        setAssets(referenceAssets);
        setSourceComparisons(nextSourceComparisons);
        setCandidateAssetId((selected) =>
          referenceAssets.some((asset) => asset.id === selected)
            ? selected
            : (referenceAssets[0]?.id ?? ""),
        );
        setCurrentId((selected) =>
          nextReceipts.some((receipt) => receipt.id === selected)
            ? selected
            : (nextReceipts[0]?.id ?? ""),
        );
        setActiveComparisonId((selected) =>
          nextSourceComparisons.some((comparison) => comparison.comparisonId === selected)
            ? selected
            : (nextSourceComparisons[0]?.comparisonId ?? ""),
        );
      })
      .catch((cause: unknown) => {
        if (requestId.current === sequence) {
          setError(cause instanceof Error ? cause.message : "Could not load capture history.");
        }
      })
      .finally(() => {
        if (requestId.current === sequence) setLoading(false);
      });
    return () => {
      requestId.current += 1;
    };
  }, [workspaceId, stateId]);

  const current = receipts.find((receipt) => receipt.id === currentId);
  const activeComparison = sourceComparisons.find(
    (comparison) => comparison.comparisonId === activeComparisonId,
  );
  const sourceImage = activeComparison
    ? assets.find((asset) => asset.id === activeComparison.sourceAssetId)
    : assets.find((asset) => asset.id === sourceBaselines.accepted?.assetId);
  const comparisonReceipt = activeComparison
    ? receipts.find((receipt) => receipt.id === activeComparison.currentReceiptId)
    : current;

  async function compare() {
    if (!stateId || !currentId || !sourceBaselines.accepted) return;
    const sequence = requestId.current;
    setComparing(true);
    setError(null);
    try {
      const created = await compareWorkspaceSourceBaseline(workspaceId, stateId, currentId);
      if (requestId.current !== sequence) return;
      setSourceComparisons((previous) => [
        created,
        ...previous.filter((item) => item.comparisonId !== created.comparisonId),
      ]);
      setActiveComparisonId(created.comparisonId);
    } catch (cause) {
      if (requestId.current === sequence)
        setError(cause instanceof Error ? cause.message : "Could not compare these captures.");
    } finally {
      if (requestId.current === sequence) setComparing(false);
    }
  }

  async function addCandidate() {
    if (!stateId || !candidateAssetId) return;
    const sequence = requestId.current;
    setError(null);
    try {
      await createWorkspaceSourceBaseline(workspaceId, stateId, candidateAssetId);
      const next = await listWorkspaceSourceBaselines(workspaceId, stateId);
      if (requestId.current === sequence) setSourceBaselines(next);
    } catch (cause) {
      if (requestId.current === sequence)
        setError(cause instanceof Error ? cause.message : "Could not add source candidate.");
    }
  }

  async function acceptCandidate(id: string) {
    if (!stateId || !acceptanceLabel.trim()) return;
    const sequence = requestId.current;
    setError(null);
    try {
      await acceptWorkspaceSourceBaseline(workspaceId, stateId, id, acceptanceLabel.trim());
      const next = await listWorkspaceSourceBaselines(workspaceId, stateId);
      if (requestId.current === sequence) {
        setSourceBaselines(next);
        setAcceptanceLabel("");
      }
    } catch (cause) {
      if (requestId.current === sequence)
        setError(cause instanceof Error ? cause.message : "Could not accept source baseline.");
    }
  }

  function captureLabel(receipt: WorkspaceCaptureReceipt): string {
    const date = new Date(receipt.capturedAt).toLocaleString();
    return `${t("workbench.workspace.revision", { defaultValue: "Revision" })} ${receipt.sourceRevision} · ${date}`;
  }

  return (
    <section className="workspace-drift" aria-labelledby="workspace-drift-title">
      <div className="workspace-drift-heading">
        <div>
          <h2 id="workspace-drift-title">
            {t("workbench.workspace.visualDrift", { defaultValue: "Visual drift" })}
          </h2>
          <p>
            {t("workbench.workspace.visualDriftHelp", {
              defaultValue:
                "Compare immutable captures for this state. This does not change component coverage.",
            })}
          </p>
        </div>
        <button
          type="button"
          className="workspace-drift-compare"
          disabled={!sourceBaselines.accepted || !currentId || comparing || loading}
          onClick={() => void compare()}
          aria-busy={comparing}
        >
          {comparing
            ? t(strings.workbench.workspace.comparing)
            : t(strings.workbench.workspace.sourceCompareButton)}
        </button>
      </div>

      {error && (
        <p className="workspace-drift-error" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="workspace-drift-empty">
          {t("workbench.workspace.loadingCaptureHistory", {
            defaultValue: "Loading capture history…",
          })}
        </p>
      ) : (
        <>
          {receipts.length === 0 && (
            <p className="workspace-drift-empty">
              {t(strings.workbench.workspace.sourceNoReceipt)}
            </p>
          )}
          <div className="workspace-drift-source">
            <h3>{t(strings.workbench.workspace.sourceBaselineHeading)}</h3>
            {sourceBaselines.accepted ? (
              <p>
                <strong>{t(strings.workbench.workspace.sourceAccepted)}</strong>:{" "}
                {assets.find((asset) => asset.id === sourceBaselines.accepted?.assetId)?.name ??
                  (sourceBaselines.accepted.sourceLocator || sourceBaselines.accepted.assetId)}{" "}
                ·{" "}
                {t(strings.workbench.workspace.sourceReviewerAttribution, {
                  label: sourceBaselines.accepted.acceptedBy,
                })}
              </p>
            ) : (
              <p>{t(strings.workbench.workspace.sourceNoAccepted)}</p>
            )}
            {assets.length > 0 && (
              <div className="workspace-drift-source-action">
                <label>
                  <span>{t(strings.workbench.workspace.sourceReferenceCandidate)}</span>
                  <select
                    aria-label={t(strings.workbench.workspace.sourceReferenceCandidate)}
                    value={candidateAssetId}
                    onChange={(event) => setCandidateAssetId(event.target.value)}
                  >
                    {assets.map((asset) => (
                      <option key={asset.id} value={asset.id}>
                        {asset.name}
                      </option>
                    ))}
                  </select>
                </label>
                <button
                  type="button"
                  onClick={() => void addCandidate()}
                  disabled={!candidateAssetId}
                >
                  {t(strings.workbench.workspace.sourceAddCandidate)}
                </button>
              </div>
            )}
            {sourceBaselines.candidates.map((candidate) => (
              <div className="workspace-drift-candidate" key={candidate.id}>
                <img
                  src={workspaceAssetUrl({
                    url: assets.find((asset) => asset.id === candidate.assetId)?.url ?? "",
                  })}
                  alt=""
                />
                <span>
                  <strong>{t(strings.workbench.workspace.sourceCandidateNotAccepted)}</strong> ·{" "}
                  {assets.find((asset) => asset.id === candidate.assetId)?.name ??
                    (candidate.sourceLocator || candidate.assetId)}
                </span>
                <label>
                  <span>{t(strings.workbench.workspace.sourceAcceptanceLabel)}</span>
                  <input
                    value={acceptanceLabel}
                    onChange={(event) => setAcceptanceLabel(event.target.value)}
                    placeholder={t(strings.workbench.workspace.sourceAcceptancePlaceholder)}
                  />
                </label>
                <button
                  type="button"
                  disabled={!acceptanceLabel.trim()}
                  onClick={() => void acceptCandidate(candidate.id)}
                >
                  {t(strings.workbench.workspace.sourceAccept)}
                </button>
              </div>
            ))}
            <small>{t(strings.workbench.workspace.sourceAcceptanceProvenance)}</small>
          </div>
          <div className="workspace-drift-selectors">
            <label>
              <span>{t(strings.workbench.workspace.sourceCurrentCapture)}</span>
              <select value={currentId} onChange={(event) => setCurrentId(event.target.value)}>
                {receipts.map((receipt) => (
                  <option key={receipt.id} value={receipt.id}>
                    {captureLabel(receipt)}
                  </option>
                ))}
              </select>
            </label>
            {sourceComparisons.length > 0 && (
              <label>
                <span>{t(strings.workbench.workspace.sourceComparisonHistory)}</span>
                <select
                  value={activeComparisonId}
                  onChange={(event) => setActiveComparisonId(event.target.value)}
                >
                  {sourceComparisons.map((comparison) => (
                    <option key={comparison.comparisonId} value={comparison.comparisonId}>
                      {t(strings.workbench.workspace.sourceComparisonLabel, {
                        date: new Date(comparison.createdAt).toLocaleString(),
                        baseline: comparison.sourceBaselineId.slice(0, 8),
                        receipt: comparison.currentReceiptId.slice(0, 8),
                      })}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </div>

          <div className="workspace-drift-images">
            {sourceImage && (
              <figure>
                <figcaption>
                  {activeComparison
                    ? t(strings.workbench.workspace.sourceComparisonSourceImage)
                    : t(strings.workbench.workspace.sourceAcceptedImage)}
                </figcaption>
                <img
                  src={workspaceAssetUrl({ url: sourceImage.url })}
                  alt={
                    activeComparison
                      ? t(strings.workbench.workspace.sourceComparisonSourceImage)
                      : t(strings.workbench.workspace.sourceAcceptedImage)
                  }
                />
              </figure>
            )}
            {comparisonReceipt && (
              <figure>
                <figcaption>
                  {activeComparison
                    ? t(strings.workbench.workspace.sourceComparisonCaptureImage)
                    : t("workbench.workspace.current", { defaultValue: "Current" })}
                </figcaption>
                <img
                  src={workspaceCaptureScreenshotUrl(
                    workspaceId,
                    stateId ?? "",
                    comparisonReceipt.id,
                  )}
                  alt={captureLabel(comparisonReceipt)}
                />
              </figure>
            )}
            {activeComparison && (
              <figure>
                <figcaption>{t(strings.workbench.workspace.sourceHeatmapCaption)}</figcaption>
                <img
                  src={workspaceSourceVisualHeatmapUrl(
                    workspaceId,
                    stateId ?? "",
                    activeComparison.comparisonId,
                  )}
                  alt={t(strings.workbench.workspace.sourceHeatmapAlt)}
                />
              </figure>
            )}
          </div>

          {activeComparison && (
            <div className="workspace-drift-metrics">
              <DescriptionList
                entries={[
                  {
                    term: t("workbench.workspace.changedPixels", {
                      defaultValue: "Changed pixels",
                    }),
                    description: `${(activeComparison.result.changedFraction * 100).toFixed(2)}%`,
                  },
                  {
                    term: t("workbench.workspace.mae", { defaultValue: "MAE" }),
                    description: activeComparison.result.mae.toFixed(4),
                  },
                  {
                    term: t("workbench.workspace.ssim", { defaultValue: "SSIM" }),
                    description: activeComparison.result.ssim.toFixed(4),
                  },
                  {
                    term: t("workbench.workspace.verdict", { defaultValue: "Verdict" }),
                    description: activeComparison.result.verdict,
                  },
                ]}
              />
            </div>
          )}
        </>
      )}
    </section>
  );
}
