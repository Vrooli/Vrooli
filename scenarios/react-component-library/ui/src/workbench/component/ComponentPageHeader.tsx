import { Link } from "react-router-dom";
import { Layers, Play, RefreshCw, Share2, Sparkles } from "lucide-react";
import { useTranslation } from "../../i18n";
import { assetAdoptions, type WorkbenchStory } from "../data";
import type { CatalogAsset } from "../../api/components";
import { appRoutes } from "../../routes";
import { evidenceStatusCopy, evidenceStatusTone, type EvidenceStatus } from "../experienceSummary";
import { Chip, Dot, Select, Segmented, WbButton } from "../primitives";
import type { DrawerTab } from "./AssetDataDrawer";
import type { CheckSummary } from "./contextChecks";
type Mode = "fixtures" | "context";
interface Props {
  asset: CatalogAsset;
  latestVersion: string;
  mode: Mode;
  stories: WorkbenchStory[];
  selectedStory?: WorkbenchStory;
  copied: boolean;
  openingWorkspace: boolean;
  canOpenWorkspace: boolean;
  runningEvidence: boolean;
  evidenceStatus: EvidenceStatus;
  fitSummary: CheckSummary;
  pageIssues: number;
  onShare: () => void;
  onOpenWorkspace: () => void;
  onRunEvidence: () => void;
  onOpenDrawer: (tab: DrawerTab) => void;
  onChooseMode: (mode: Mode) => void;
  onChooseStory: (story: WorkbenchStory) => void;
  onRerunStory: () => void;
  onToggleDrawer: () => void;
}
function FitSummary({ summary }: { summary: CheckSummary }) {
  const { t } = useTranslation();
  const parts: string[] = [];
  if (summary.fail > 0) parts.push(`${summary.fail} fail`);
  if (summary.warn > 0) parts.push(`${summary.warn} warn`);
  if (summary.unmeasured > 0) parts.push(`${summary.unmeasured} n/m`);
  if (summary.fail === 0 && summary.warn === 0) {
    if (summary.pass > 0) parts.unshift(`${summary.pass} pass`);
    else if (parts.length === 0) {
      parts.push(t("workbench.component.noChecks", { defaultValue: "No checks" }));
    }
  }
  const tone =
    summary.fail > 0 ? "fail" : summary.warn > 0 ? "warn" : summary.unmeasured > 0 ? "nm" : "pass";
  return (
    <span
      className="wb-fitline"
      data-testid="fit-summary"
      title={t("workbench.component.fitHint", {
        defaultValue: "Component fit, measured on this component.",
      })}
    >
      <i className={`wb-summary-dot is-${tone}`} aria-hidden />
      <b>{t("workbench.component.fit", { defaultValue: "Fit" })}</b>
      <span>{parts.join(" · ")}</span>
    </span>
  );
}

export function ComponentPageHeader({
  asset,
  latestVersion,
  mode,
  stories,
  selectedStory,
  copied,
  openingWorkspace,
  canOpenWorkspace,
  runningEvidence,
  evidenceStatus,
  fitSummary,
  pageIssues,
  onShare,
  onOpenWorkspace,
  onRunEvidence,
  onOpenDrawer,
  onChooseMode,
  onChooseStory,
  onRerunStory,
  onToggleDrawer,
}: Props) {
  const { t } = useTranslation();
  const breadcrumb = (asset.catalogId || asset.libraryId).split(".").filter(Boolean);
  const statusCopy = evidenceStatusCopy(evidenceStatus);
  const statusTone = evidenceStatusTone(evidenceStatus);
  return (
    <>
      <header className="wb-comp-head">
        <div className="wb-row" style={{ flexWrap: "wrap", gap: 6 }}>
          <Link to={appRoutes.components} className="wb-muted wb-small">
            {t("workbench.component.breadcrumbComponents", { defaultValue: "Components" })}
          </Link>
          {breadcrumb.map((part) => (
            <span key={part} className="wb-row" style={{ gap: 6 }}>
              <span className="wb-muted-2">/</span>
              <span className="wb-muted wb-small">{part}</span>
            </span>
          ))}
        </div>

        <div className="wb-comp-title-row">
          <div style={{ minWidth: 0 }}>
            <h1 className="wb-comp-title">{asset.displayName}</h1>
            <p className="wb-comp-desc">{asset.description}</p>
          </div>
          <div className="wb-actions">
            <WbButton variant="ghost" size="md" onClick={onShare}>
              <Share2 aria-hidden size={14} />
              {copied
                ? t("workbench.component.copied", { defaultValue: "Copied" })
                : t("workbench.component.share", { defaultValue: "Share" })}
            </WbButton>
            <WbButton
              variant="ghost"
              size="md"
              onClick={onOpenWorkspace}
              disabled={openingWorkspace || !canOpenWorkspace}
            >
              <Sparkles aria-hidden size={14} />
              {t("workbench.component.useInDesign", { defaultValue: "Use in design" })}
            </WbButton>
            <WbButton
              variant="primary"
              size="md"
              onClick={onRunEvidence}
              disabled={runningEvidence}
            >
              <Play aria-hidden size={14} />
              {runningEvidence
                ? t("workbench.component.running", { defaultValue: "Running…" })
                : t("workbench.component.runEvidence", { defaultValue: "Run evidence" })}
            </WbButton>
          </div>
        </div>

        <div className="wb-comp-meta">
          <Chip tone="ice">{`v${latestVersion}`}</Chip>
          <Chip>{asset.slot || t("workbench.component.noSlot", { defaultValue: "no slot" })}</Chip>
          <Chip>
            {t("workbench.component.direct", {
              defaultValue: "{{count}} direct",
              count: assetAdoptions(asset),
            })}
          </Chip>
          <Chip>
            {t("workbench.component.effective", {
              defaultValue: "{{count}} transitive",
              count: asset.metrics?.effectiveAdoptionCount ?? 0,
            })}
          </Chip>
          <Chip tone={statusTone}>
            <Dot tone={statusTone} />
            {t(statusCopy.key, { defaultValue: statusCopy.defaultValue })}
          </Chip>
          <span className="wb-muted-2">·</span>
          <button type="button" className="wb-link" onClick={() => onOpenDrawer("source")}>
            {t("workbench.component.source", { defaultValue: "Source" })}
          </button>
          <button type="button" className="wb-link" onClick={() => onOpenDrawer("versions")}>
            {t("workbench.component.versions", { defaultValue: "Versions" })}
          </button>
          <button type="button" className="wb-link" onClick={() => onOpenDrawer("experience")}>
            {t("workbench.component.experience", { defaultValue: "Experience" })}
          </button>
          <button type="button" className="wb-link" onClick={() => onOpenDrawer("tests")}>
            {t("workbench.component.tests", { defaultValue: "Tests" })}
          </button>
          <button type="button" className="wb-link" onClick={() => onOpenDrawer("relationships")}>
            {t("workbench.component.relationships", { defaultValue: "Relationships" })}
          </button>
        </div>
      </header>

      <div className="wb-modebar">
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <Segmented
            label={t("workbench.component.mode", { defaultValue: "Component mode" })}
            value={mode}
            onChange={onChooseMode}
            items={[
              {
                id: "fixtures",
                label: t("workbench.component.fixtures", { defaultValue: "Fixtures" }),
              },
              {
                id: "context",
                label: t("workbench.component.inContext", { defaultValue: "In context" }),
              },
            ]}
          />
          <Select
            aria-label={t("workbench.component.story", { defaultValue: "Story" })}
            value={selectedStory?.id ?? ""}
            onChange={(event) => {
              const story = stories.find((entry) => entry.id === event.target.value);
              if (story) onChooseStory(story);
            }}
            style={{ width: "auto", minWidth: 170 }}
          >
            {stories.length === 0 ? (
              <option value="">
                {t("workbench.component.noStory", { defaultValue: "No stories" })}
              </option>
            ) : null}
            {stories.map((story) => (
              <option key={story.id} value={story.id}>
                {story.name || story.id}
              </option>
            ))}
          </Select>
          <button type="button" className="wb-btn ghost sm" onClick={onRerunStory}>
            <RefreshCw aria-hidden size={13} />
            {t("workbench.component.rerun", { defaultValue: "Re-run" })}
          </button>
        </div>
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <FitSummary summary={fitSummary} />
          {pageIssues > 0 ? (
            <span
              className="wb-page-issue"
              title={t("workbench.component.pageIssueHint", {
                defaultValue: "Page health, not this component.",
              })}
            >
              <i className="wb-summary-dot is-fail" aria-hidden />
              {t("workbench.component.pageIssueCount", {
                defaultValue: "{{count}} page issue",
                count: pageIssues,
              })}
            </span>
          ) : null}
          <button type="button" className="wb-btn ghost sm" onClick={onToggleDrawer}>
            <Layers aria-hidden size={13} />
            {t("workbench.component.assetData", { defaultValue: "Asset data" })}
          </button>
        </div>
      </div>
    </>
  );
}
