import { Box, Frame as FrameIcon } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../designs";
import { layerLabel, type TemplateNode, type TemplatePalette } from "../templates/spec";
import type { FrameAuditMap } from "./checks";
import { TOUCH_TARGET_MIN, type StateAudit } from "./checks";
import { FRAME_SIZE, paletteTokens } from "./model";
import { layerCount, rclPrimitive, type Coverage } from "./rcl";
import { Chip, Empty, Tabs } from "../primitives";
import { NodeInspector } from "./NodeInspector";
import { cx } from "../cx";
import { VERDICT_ICON, VERDICT_KEY, type Verdict } from "../verdicts";
import type { StudioProposal } from "./proposal";
import { StudioAssist, type AssistScope } from "./StudioAssist";

export type InspectorTab = "inspect" | "checks" | "ai";
export type AiScope = AssistScope;

export interface StudioClaim {
  id: string;
  label: string;
  detail: string;
  verdict: Verdict;
}

function VerdictRow({ verdict, label, detail }: { verdict: Verdict; label: string; detail: string }) {
  const { t } = useTranslation();
  const Icon = VERDICT_ICON[verdict];
  return (
    <div className={cx("studio-check", `is-${verdict}`)}>
      <span className="studio-check-mark" aria-hidden>
        <Icon size={12} />
      </span>
      <span className="studio-check-copy">
        <b>{label}</b>
        <small>{detail}</small>
      </span>
      <span className="studio-check-verdict">
        {t(VERDICT_KEY[verdict], { defaultValue: verdict })}
      </span>
    </div>
  );
}

export function StudioInspector({
  tab,
  onTab,
  design,
  frame,
  theme,
  selected,
  onPropChange,
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
}: {
  tab: InspectorTab;
  onTab: (tab: InspectorTab) => void;
  design: DesignDocument;
  frame: DesignFrame;
  theme: TemplatePalette;
  selected: TemplateNode | null;
  onPropChange: (key: string, value: string | number | boolean) => void;
  coverage: Coverage;
  claims: StudioClaim[];
  frames: DesignFrame[];
  audits: FrameAuditMap;
  stateGap: StateAudit;
  aiScope: AiScope;
  onAiScope: (scope: AiScope) => void;
  onSelectFrame: (id: string) => void;
  onApplyProposal: (proposal: StudioProposal) => { ok: true } | { ok: false; reason: "stale" | "invalid"; detail: string };
  onApplied: (summary: string) => void;
}) {
  const { t } = useTranslation();
  const size = FRAME_SIZE[frame.device];
  const primitive = selected ? rclPrimitive(selected.type) : null;
  const overflowFrames = frames.filter((entry) => {
    const audit = audits[entry.id];
    return Boolean(audit?.measured && audit.overflow > 0);
  });
  const measuredFrames = frames.filter((entry) => audits[entry.id]?.measured);
  const targetAudits = frames.flatMap((entry) => (audits[entry.id]?.buttons ? [audits[entry.id]] : []));
  const targetTotal = targetAudits.reduce((sum, audit) => sum + (audit?.buttons?.total ?? 0), 0);
  const targetUnder = targetAudits.reduce((sum, audit) => sum + (audit?.buttons?.under ?? 0), 0);
  const overflowVerdict = measuredFrames.length === 0 ? "unmeasured" : overflowFrames.length > 0 ? "fail" : "pass";
  const targetVerdict = targetTotal === 0 ? "unmeasured" : targetUnder > 0 ? "warn" : "pass";
  const stateVerdict = stateGap.missing.length > 0 ? "warn" : "pass";
  const issueCount =
    claims.filter((claim) => claim.verdict === "warn" || claim.verdict === "fail").length +
    (overflowFrames.length > 0 ? 1 : 0) +
    (targetUnder > 0 ? 1 : 0) +
    (stateGap.missing.length > 0 ? 1 : 0);
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
        {selected ? <Chip tone={primitive ? "green" : undefined}>{selected.type}</Chip> : null}
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
            { id: "ai", label: t("workbench.studio.ai", { defaultValue: "AI" }) },
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
          <>
            <div className="studio-inspector-section">
              <div className="studio-score">
                <span>
                  <b>{claims.length + 3 - issueCount}</b>
                  <small>
                    {t("workbench.studio.checksPassing", {
                      defaultValue: "{{passed}} of {{total}} checks pass",
                      passed: claims.length + 3 - issueCount,
                      total: claims.length + 3,
                    })}
                  </small>
                </span>
                <span className="studio-score-bar" aria-hidden>
                  <i
                    style={{
                      width: `${Math.round(((claims.length + 3 - issueCount) / (claims.length + 3)) * 100)}%`,
                    }}
                  />
                </span>
              </div>
            </div>

            <div className="studio-inspector-section">
              <span className="wb-caption">
                {t("workbench.studio.layoutChecks", { defaultValue: "Layout" })}
              </span>
              <VerdictRow
                verdict={overflowVerdict}
                label={t("workbench.studio.checkOverflow", {
                  defaultValue: "No horizontal overflow",
                })}
                detail={
                  overflowFrames.length === 0
                    ? t("workbench.studio.checkOverflowPass", {
                        defaultValue: "Measured clean at {{count}} frames",
                        count: frames.filter((entry) => audits[entry.id]?.measured).length,
                      })
                    : t("workbench.studio.checkOverflowFail", {
                        defaultValue: "{{names}} overflow",
                        names: overflowFrames.map((entry) => entry.name).join(", "),
                      })
                }
              />
              <VerdictRow
                verdict={targetVerdict}
                label={t("workbench.studio.checkTargets", {
                  defaultValue: "Touch targets ≥ {{min}}",
                  min: TOUCH_TARGET_MIN,
                })}
                detail={
                  targetTotal === 0
                    ? t("workbench.studio.checkTargetsUnmeasured", {
                        defaultValue: "No interactive elements measured yet",
                      })
                    : t("workbench.studio.checkTargetsDetail", {
                        defaultValue: "{{under}} of {{total}} buttons under {{min}}",
                        under: targetUnder,
                        total: targetTotal,
                        min: TOUCH_TARGET_MIN,
                      })
                }
              />
              <VerdictRow
                verdict={stateVerdict}
                label={t("workbench.studio.checkStates", {
                  defaultValue: "States are declared",
                })}
                detail={
                  stateGap.missing.length === 0
                    ? t("workbench.studio.checkStatesPass", {
                        defaultValue: "{{count}} of {{count}} layers declare states",
                        count: stateGap.total,
                      })
                    : t("workbench.studio.checkStatesGap", {
                        defaultValue: "{{count}} missing · {{names}}",
                        count: stateGap.missing.length,
                        names: stateGap.missing.slice(0, 3).join(", "),
                      })
                }
              />
            </div>

            <div className="studio-inspector-section">
              <span className="wb-caption">
                {t("workbench.studio.contractChecks", { defaultValue: "Composition claims" })}
              </span>
              {claims.map((claim) => (
                <VerdictRow
                  key={claim.id}
                  verdict={claim.verdict}
                  label={claim.label}
                  detail={claim.detail}
                />
              ))}
            </div>

            <div className="studio-inspector-section">
              <p className="studio-note">
                {t("workbench.studio.checksNote", {
                  defaultValue:
                    "Layout checks measure the frames on this canvas. Contrast and capture evidence stay unmeasured until a real capture exists.",
                })}
              </p>
            </div>
          </>
        ) : null}

        {tab === "ai" ? (
          <StudioAssist
            design={design}
            frame={frame}
            selected={selected ?? undefined}
            scope={aiScope}
            onScope={onAiScope}
            onApply={onApplyProposal}
            onApplied={onApplied}
          />
        ) : null}
      </div>
    </>
  );
}
