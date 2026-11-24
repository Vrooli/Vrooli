import { Box, Frame as FrameIcon } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../design-types";
import {
  layerLabel,
  type TemplateNode,
  type TemplateNodeType,
  type TemplatePalette,
} from "../templates/spec";
import type { FrameAuditMap } from "./checks";
import type { StateAudit } from "./checks";
import { FRAME_SIZE, paletteTokens } from "./model";
import { isRclLayer, layerCount, rclPrimitive, type Coverage } from "./rcl";
import { Chip, Empty, Tabs } from "../primitives";
import { NodeInspector } from "./NodeInspector";
import { cx } from "../cx";
import type { Verdict } from "../verdicts";
import type { ProposalTransport, StudioProposal } from "./proposal";
import { StudioAssist, type AssistScope } from "./StudioAssist";
import { StudioNodeReplacement } from "./StudioNodeReplacement";
import type { CatalogAsset } from "../data";
import { StudioInspectorChecks } from "./StudioInspectorChecks";
import { buildStudioInspectorCheckSummary } from "./studioInspectorCheckSummary";

export type InspectorTab = "inspect" | "checks" | "ai";
export type AiScope = AssistScope;

export interface StudioClaim {
  id: string;
  label: string;
  detail: string;
  verdict: Verdict;
}

export function StudioInspector({
  tab,
  onTab,
  design,
  frame,
  theme,
  selected,
  onPropChange,
  onReplaceNode,
  catalogCandidates,
  catalogLoading,
  catalogFailed,
  pendingCatalogAssetId,
  onReplaceCatalogAsset,
  coverage,
  claims,
  frames,
  audits,
  stateGap,
  aiScope,
  onAiScope,
  onSelectFrame,
  onApplyProposal,
  onApplied,
  assist = true,
  propose,
}: {
  tab: InspectorTab;
  onTab: (tab: InspectorTab) => void;
  design: DesignDocument;
  frame: DesignFrame;
  theme: TemplatePalette;
  selected: TemplateNode | null;
  onPropChange: (key: string, value: string | number | boolean) => void;
  onReplaceNode: (type: TemplateNodeType) => void;
  catalogCandidates: readonly CatalogAsset[];
  catalogLoading: boolean;
  catalogFailed: boolean;
  pendingCatalogAssetId: string;
  onReplaceCatalogAsset: (componentId: string) => void;
  coverage: Coverage;
  claims: StudioClaim[];
  frames: DesignFrame[];
  audits: FrameAuditMap;
  stateGap: StateAudit;
  aiScope: AiScope;
  onAiScope: (scope: AiScope) => void;
  onSelectFrame: (id: string) => void;
  onApplyProposal: (
    proposal: StudioProposal,
  ) => { ok: true } | { ok: false; reason: "stale" | "invalid"; detail: string };
  onApplied: (summary: string) => void;
  /**
   * AI assist is design-store backed. A surface that has no proposal endpoint
   * hides the tab instead of presenting a control that cannot work.
   */
  assist?: boolean;
  /** Required when assist is shown; the surface owns its transport. */
  propose?: ProposalTransport;
}) {
  const { t } = useTranslation();
  const size = FRAME_SIZE[frame.device];
  const boundComponent =
    selected?.type === "slot" && typeof selected.props.componentId === "string"
      ? catalogCandidates.find((asset) => asset.id === selected.props.componentId)
      : undefined;
  const boundCatalogSlot = Boolean(selected && selected.type === "slot" && isRclLayer(selected));
  const primitive = selected
    ? (rclPrimitive(selected.type) ??
      (boundCatalogSlot
        ? boundComponent?.displayName ||
          t("workbench.studio.catalogComponent", { defaultValue: "Catalog component" })
        : null))
    : null;
  const checkSummary = buildStudioInspectorCheckSummary({
    coverage,
    audits,
    frames,
    stateGap,
    selected,
    claims,
  });
  const issueCount = checkSummary.checks.needsReview;
  const tokenPreview = paletteTokens(theme).filter((token) =>
    ["bg", "ink", "line", "accent"].includes(token.key),
  );

  return (
    <>
      <div className="studio-pane-head">
        <span className="wb-caption">
          {t("workbench.studio.selection", { defaultValue: "Selection" })}
        </span>
        <span className="studio-frame-chip">{frame.name}</span>
      </div>

      <div className="studio-inspector-head">
        <span className="studio-inspector-icon" aria-hidden>
          {selected ? <Box size={14} /> : <FrameIcon size={14} />}
        </span>
        <span className="studio-inspector-title">
          <b>
            {selected
              ? layerLabel(selected)
              : t("workbench.studio.frameSelected", {
                  defaultValue: "{{name}} frame",
                  name: frame.name,
                })}
          </b>
          <small>
            {selected
              ? primitive
                ? t("workbench.studio.rclPrimitive", {
                    defaultValue: "RCL · {{name}}",
                    name: primitive,
                  })
                : t("workbench.studio.customLayer", { defaultValue: "Custom layer" })
              : t("workbench.studio.frameMeta", {
                  defaultValue: "{{device}} · {{width}} × {{height}}",
                  device: frame.device,
                  width: size.width,
                  height: size.height,
                })}
          </small>
        </span>
        {selected ? (
          <Chip tone={primitive ? "green" : undefined}>
            {boundCatalogSlot ? primitive : selected.type}
          </Chip>
        ) : null}
      </div>

      <div className="studio-inspector-tabs">
        <Tabs
          label={t("workbench.studio.inspectorTabs", { defaultValue: "Inspector sections" })}
          value={tab}
          onChange={onTab}
          items={[
            { id: "inspect", label: t("workbench.studio.inspect", { defaultValue: "Inspect" }) },
            {
              id: "checks",
              label: t("workbench.studio.checks", { defaultValue: "Checks" }),
              count: issueCount,
            },
            ...(assist
              ? [{ id: "ai" as const, label: t("workbench.studio.ai", { defaultValue: "AI" }) }]
              : []),
          ]}
        />
      </div>

      <div className="studio-inspector-body">
        {tab === "inspect" ? (
          selected ? (
            <>
              <div className="studio-inspector-section">
                <NodeInspector node={selected} onChange={onPropChange} />
              </div>
              <div className="studio-inspector-section">
                <StudioNodeReplacement
                  node={selected}
                  onReplace={onReplaceNode}
                  catalogCandidates={catalogCandidates}
                  catalogLoading={catalogLoading}
                  catalogFailed={catalogFailed}
                  pendingCatalogAssetId={pendingCatalogAssetId}
                  onReplaceCatalogAsset={onReplaceCatalogAsset}
                />
              </div>
              <div className="studio-inspector-section">
                <span className="wb-caption">
                  {t("workbench.studio.systemPath", { defaultValue: "System path" })}
                </span>
                <div className="studio-kv">
                  <span>{t("workbench.studio.primitive", { defaultValue: "primitive" })}</span>
                  <b>
                    {primitive ??
                      t("workbench.studio.none", { defaultValue: "custom composition" })}
                  </b>
                </div>
                <div className="studio-kv">
                  <span>{t("workbench.studio.tokensUsed", { defaultValue: "tokens" })}</span>
                  <b>
                    {t("workbench.studio.tokenPair", {
                      defaultValue: "{{count}} roles",
                      count: tokenPreview.length,
                    })}
                  </b>
                </div>
                <div className="studio-token-strip">
                  {tokenPreview.map((token) => (
                    <span key={token.key} className="studio-token-chip">
                      <span style={{ background: token.value }} aria-hidden />
                      {token.key}
                    </span>
                  ))}
                </div>
              </div>
            </>
          ) : (
            <div className="studio-inspector-section">
              <Empty title={t("workbench.studio.selectLayer", { defaultValue: "Select a layer" })}>
                {t("workbench.studio.selectLayerBody", {
                  defaultValue: "Choose a layer in the tree or click an element in a frame.",
                })}
              </Empty>
            </div>
          )
        ) : null}

        {tab === "inspect" ? (
          <div className="studio-inspector-section">
            <span className="wb-caption">
              {t("workbench.studio.frameSummary", { defaultValue: "Frame" })}
            </span>
            <div className="studio-kv">
              <span>{t("workbench.studio.device", { defaultValue: "Device" })}</span>
              <b>{frame.device}</b>
            </div>
            <div className="studio-kv">
              <span>{t("workbench.studio.size", { defaultValue: "Size" })}</span>
              <b>
                {size.width} × {size.height}
              </b>
            </div>
            <div className="studio-kv">
              <span>{t("workbench.studio.layers", { defaultValue: "Layers" })}</span>
              <b>{layerCount(frame.spec)}</b>
            </div>
            <div className="studio-kv">
              <span>{t("workbench.studio.rclShare", { defaultValue: "RCL share" })}</span>
              <b>{Math.round(coverage.ratio * 100)}%</b>
            </div>
            <div className="studio-token-strip">
              {frames.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  className={cx("studio-chip-btn", entry.id === frame.id && "is-active")}
                  aria-pressed={entry.id === frame.id}
                  onClick={() => onSelectFrame(entry.id)}
                >
                  {entry.name}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        {tab === "checks" ? (
          <StudioInspectorChecks
            summary={checkSummary}
            claims={claims}
            stateGap={stateGap}
            selected={selected}
            onSelectFrame={onSelectFrame}
            onTab={onTab}
            onReplaceNode={onReplaceNode}
          />
        ) : null}

        {assist && propose && tab === "ai" ? (
          <StudioAssist
            design={design}
            frame={frame}
            selected={selected ?? undefined}
            scope={aiScope}
            onScope={onAiScope}
            onApply={onApplyProposal}
            onApplied={onApplied}
            propose={propose}
          />
        ) : null}
      </div>
    </>
  );
}
