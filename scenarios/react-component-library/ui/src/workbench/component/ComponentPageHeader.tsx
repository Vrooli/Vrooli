import { Link } from "react-router-dom";
import { Layers, Play, RefreshCw, Share2, Sparkles } from "lucide-react";
import { useTranslation } from "../../i18n";
import { assetAdoptions, type WorkbenchStory } from "../data";
import type { CatalogAsset, ComponentExperience } from "../../api/components";
import { appRoutes } from "../../routes";
import { Chip, Dot, Select, Segmented, WbButton } from "../primitives";
import type { DrawerTab } from "./AssetDataDrawer";
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
  experience?: ComponentExperience;
  checksCount: number;
  passed: number;
  checkRatio: number;
  onShare: () => void;
  onOpenWorkspace: () => void;
  onRunEvidence: () => void;
  onOpenDrawer: (tab: DrawerTab) => void;
  onChooseMode: (mode: Mode) => void;
  onChooseStory: (story: WorkbenchStory) => void;
  onRerunStory: () => void;
  onToggleDrawer: () => void;
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
  experience,
  checksCount,
  passed,
  checkRatio,
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
          {experience ? (
            <Chip tone={experience.evidenceStatus === "fresh" ? "green" : "amber"}>
              <Dot tone={experience.evidenceStatus === "fresh" ? "green" : "amber"} />
              {experience.evidenceStatus}
            </Chip>
          ) : (
            <Chip tone="amber">
              {t("workbench.component.unmeasured", { defaultValue: "unmeasured" })}
            </Chip>
          )}
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
          <Chip tone={checkRatio === 1 && checksCount > 0 ? "green" : "amber"}>
            {checksCount === 0
              ? t("workbench.component.noChecks", { defaultValue: "No checks" })
              : t("workbench.component.checksSummary", {
                  defaultValue: "{{passed}}/{{total}} claims pass",
                  passed,
                  total: checksCount,
                })}
          </Chip>
          <button type="button" className="wb-btn ghost sm" onClick={onToggleDrawer}>
            <Layers aria-hidden size={13} />
            {t("workbench.component.assetData", { defaultValue: "Asset data" })}
          </button>
        </div>
      </div>
    </>
  );
}
