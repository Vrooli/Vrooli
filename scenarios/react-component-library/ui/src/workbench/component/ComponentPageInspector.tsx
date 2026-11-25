import type { Dispatch, SetStateAction } from "react";
import { Link } from "react-router-dom";
import { appRoutes } from "../../routes";
import { ArrowRight, ArrowUpRight } from "lucide-react";
import { useTranslation } from "../../i18n";
import type { CatalogAsset } from "../../api/components";
import type { WorkbenchStory } from "../data";
import { formatRelative, assetAdoptions } from "../data";
import type { InContextReport } from "./InContextStage";
import type { Template } from "../templates/spec";
import { InContextChecksPanel } from "./InContextChecks";
import { ChecksList, PropsEditor } from "./ComponentInspectorEditors";
import { Ring, Tabs, WbButton } from "../primitives";

type Mode = "fixtures" | "context";
type InspectorTab = "checks" | "props" | "usage";
type Check = { claim: { id: string; statement: string }; verdict: string; checkedAt?: string };
interface Props {
  asset: CatalogAsset;
  versionsCount: number;
  mode: Mode;
  selectedStory?: WorkbenchStory;
  checks: Check[];
  checkRatio: number;
  contextRatio: number;
  contextScored: number;
  contextRows: InContextReport["rows"];
  contextReport: InContextReport | null;
  inspectorTab: InspectorTab;
  setInspectorTab: (tab: InspectorTab) => void;
  overrides: Record<string, Record<string, unknown> | null>;
  setOverrides: Dispatch<SetStateAction<Record<string, Record<string, unknown> | null>>>;
  statuses: Record<string, "loading" | "ready" | "error" | "applied">;
  template: Template | null;
  openingWorkspace: boolean;
  canOpenWorkspace: boolean;
  onOpenWorkspace: () => void;
}
export function ComponentPageInspector({
  asset,
  versionsCount,
  mode,
  selectedStory,
  checks,
  checkRatio,
  contextRatio,
  contextScored,
  contextRows,
  contextReport,
  inspectorTab,
  setInspectorTab,
  overrides,
  setOverrides,
  statuses,
  template,
  openingWorkspace,
  canOpenWorkspace,
  onOpenWorkspace,
}: Props) {
  const { t } = useTranslation();
  return (
    <aside
      className="wb-inspector"
      aria-label={t("workbench.component.inspector", { defaultValue: "Inspector" })}
    >
      <div className="wb-inspector-head">
        <div style={{ minWidth: 0 }}>
          <span className="wb-caption">
            {mode === "fixtures"
              ? t("workbench.component.selectedFixture", { defaultValue: "Selected fixture" })
              : t("workbench.component.inContextSettings", {
                  defaultValue: "In-context settings",
                })}
          </span>
          <h2>
            {mode === "fixtures"
              ? selectedStory?.name || selectedStory?.id || "—"
              : asset.displayName}
          </h2>
        </div>
        <Ring
          value={mode === "fixtures" ? checkRatio * 100 : contextRatio * 100}
          size={58}
          tone={
            (mode === "fixtures" ? checkRatio : contextRatio) >= 1 &&
            (mode === "fixtures" ? checks.length : contextScored) > 0
              ? "green"
              : "amber"
          }
        />
      </div>
      <div className="wb-inspector-tabs">
        <Tabs
          label={t("workbench.component.inspectorTabs", { defaultValue: "Inspector sections" })}
          value={inspectorTab}
          onChange={setInspectorTab}
          items={[
            {
              id: "checks",
              label: t("workbench.component.checks", { defaultValue: "Checks" }),
            },
            { id: "props", label: t("workbench.component.props", { defaultValue: "Props" }) },
            {
              id: "usage",
              label:
                mode === "fixtures"
                  ? t("workbench.component.usageTab", { defaultValue: "Usage" })
                  : t("workbench.component.pageTab", { defaultValue: "Page" }),
            },
          ]}
        />
      </div>
      <div className="wb-inspector-body">
        {inspectorTab === "checks" ? (
          mode === "fixtures" ? (
            <ChecksList
              rows={checks.map((check) => ({
                id: check.claim.id,
                label: check.claim.statement,
                verdict: check.verdict,
                detail: check.checkedAt ? formatRelative(new Date(check.checkedAt)) : undefined,
              }))}
              empty={t("workbench.component.noChecksBody", {
                defaultValue:
                  "This asset has no experience claims. Nothing is shown as a passing check.",
              })}
            />
          ) : (
            <InContextChecksPanel rows={contextRows} />
          )
        ) : null}

        {inspectorTab === "props" && selectedStory ? (
          <PropsEditor
            story={selectedStory}
            override={overrides[selectedStory.id] ?? null}
            onApply={(next) =>
              setOverrides((current) => ({ ...current, [selectedStory.id]: next }))
            }
            onReset={() => setOverrides((current) => ({ ...current, [selectedStory.id]: null }))}
            status={statuses[selectedStory.id] ?? "loading"}
          />
        ) : null}

        {inspectorTab === "usage" ? (
          mode === "fixtures" ? (
            <div className="wb-stack tight">
              <div className="wb-row-between">
                <span className="wb-caption">
                  {t("workbench.component.catalogId", { defaultValue: "Catalog id" })}
                </span>
                <span className="wb-mono wb-small">{asset.catalogId || "—"}</span>
              </div>
              <div className="wb-row-between">
                <span className="wb-caption">
                  {t("workbench.component.source", { defaultValue: "Source" })}
                </span>
                <span
                  className="wb-mono wb-small"
                  style={{ textAlign: "right", overflowWrap: "anywhere" }}
                >
                  {asset.sourcePath}
                </span>
              </div>
              <div className="wb-row-between">
                <span className="wb-caption">
                  {t("workbench.component.adoptions", { defaultValue: "Adoptions" })}
                </span>
                <span className="wb-mono wb-small">{assetAdoptions(asset)}</span>
              </div>
              <div className="wb-row-between">
                <span className="wb-caption">
                  {t("workbench.component.versionsCount", { defaultValue: "Versions" })}
                </span>
                <span className="wb-mono wb-small">{versionsCount}</span>
              </div>
              <div className="wb-divider" />
              <p className="wb-small wb-muted">
                {t("workbench.component.fixtureNote", {
                  defaultValue:
                    "Fixtures are grouped by the question they answer. Select a frame to inspect it; open the preview for a full-size view.",
                })}
              </p>
            </div>
          ) : (
            <div className="wb-stack tight">
              <span className="wb-caption">
                {t("workbench.component.templateIdea", { defaultValue: "Design idea" })}
              </span>
              <p className="wb-small wb-muted">{template?.idea ?? "Loading curated workspace…"}</p>
              <span className="wb-caption">
                {t("workbench.component.slotHint", { defaultValue: "Slot" })}
              </span>
              <p className="wb-small wb-muted">{template?.slotHint ?? ""}</p>
              <div className="wb-divider" />
              <span className="wb-caption">
                {t("workbench.component.placement", { defaultValue: "Placement" })}
              </span>
              {contextReport?.decision.chosen ? (
                <>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">
                      {t("workbench.component.ctxChosenSlot", { defaultValue: "Chosen slot" })}
                    </span>
                    <span className="wb-small">{contextReport.decision.chosen.label}</span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">
                      {t("workbench.component.ctxSlotRole", { defaultValue: "Role" })}
                    </span>
                    <span className="wb-small wb-mono">
                      {contextReport.decision.chosen.role ?? "open"}
                    </span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">
                      {t("workbench.component.ctxAssetKind", { defaultValue: "Asset kind" })}
                    </span>
                    <span className="wb-small wb-mono">{contextReport.decision.profile.kind}</span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">
                      {t("workbench.component.ctxScore", { defaultValue: "Affinity" })}
                    </span>
                    <span className="wb-small wb-mono">{contextReport.decision.chosen.score}%</span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">
                      {t("workbench.component.ctxViewport", { defaultValue: "Viewport" })}
                    </span>
                    <span className="wb-small wb-mono">
                      {contextReport.viewport} · {contextReport.device}
                    </span>
                  </div>
                </>
              ) : (
                <p className="wb-small wb-muted">
                  {t("workbench.component.ctxPlacementPending", {
                    defaultValue: "Placing the component…",
                  })}
                </p>
              )}
              <div className="wb-divider" />
              <WbButton
                variant="primary"
                size="md"
                onClick={onOpenWorkspace}
                disabled={openingWorkspace || !selectedStory || !canOpenWorkspace}
              >
                {openingWorkspace
                  ? t("workbench.component.openingWorkspace", {
                      defaultValue: "Opening workspace…",
                    })
                  : t("workbench.component.takeIntoStudio", {
                      defaultValue: "Take into studio",
                    })}
                <ArrowRight aria-hidden size={14} />
              </WbButton>
              <Link to={appRoutes.design} className="wb-btn ghost sm wide">
                {t("workbench.component.openStudio", { defaultValue: "Open design studio" })}
                <ArrowUpRight aria-hidden size={13} />
              </Link>
            </div>
          )
        ) : null}
      </div>
    </aside>
  );
}
