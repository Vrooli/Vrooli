import { useTranslation } from "../../i18n";
import { TOUCH_TARGET_MIN, type StateAudit } from "./checks";
import { cx } from "../cx";
import { VERDICT_ICON, VERDICT_KEY, type Verdict } from "../verdicts";
import type { TemplateNode, TemplateNodeType } from "../templates/spec";
import type { StudioClaim, InspectorTab } from "./StudioInspector";
import type { StudioInspectorCheckSummary } from "./studioInspectorCheckSummary";

function VerdictRow({
  verdict,
  label,
  detail,
}: {
  verdict: Verdict;
  label: string;
  detail: string;
}) {
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

export function StudioInspectorChecks({
  summary,
  claims,
  stateGap,
  selected,
  onSelectFrame,
  onTab,
  onReplaceNode,
}: {
  summary: StudioInspectorCheckSummary;
  claims: StudioClaim[];
  stateGap: StateAudit;
  selected: TemplateNode | null;
  onSelectFrame: (id: string) => void;
  onTab: (tab: InspectorTab) => void;
  onReplaceNode: (type: TemplateNodeType) => void;
}) {
  const { t } = useTranslation();
  const {
    overflowFrames,
    measuredFrameCount,
    overflowVerdict,
    targetTotal,
    targetUnder,
    targetVerdict,
    stateVerdict,
    checks,
    steps,
    bestReplacement,
  } = summary;
  return (
    <>
      <div
        className="studio-inspector-section"
        aria-label={t("workbench.studio.nextSteps", { defaultValue: "Suggested next steps" })}
      >
        <span className="wb-caption">
          {t("workbench.studio.nextSteps", { defaultValue: "Suggested next steps" })}
        </span>
        {steps.length === 0 ? (
          <p className="studio-note">
            {t("workbench.studio.noNextSteps", {
              defaultValue: "No measured issues found. Unmeasured checks remain listed below.",
            })}
          </p>
        ) : (
          steps.map((step) => (
            <div className="studio-check" key={step.id}>
              <span className="studio-check-copy">
                <b>{step.title}</b>
                <small>{step.detail}</small>
              </span>
              {step.action === "overflow-frame" && step.frameId ? (
                <button
                  type="button"
                  className="studio-chip-btn"
                  onClick={() => step.frameId && onSelectFrame(step.frameId)}
                >
                  {t("workbench.studio.openFrame", { defaultValue: "Open frame" })}
                </button>
              ) : null}
              {step.action === "states" ? (
                <button type="button" className="studio-chip-btn" onClick={() => onTab("inspect")}>
                  {t("workbench.studio.inspectLayers", { defaultValue: "Inspect layers" })}
                </button>
              ) : null}
              {step.action === "replace" && selected && bestReplacement ? (
                <button
                  type="button"
                  className="studio-chip-btn"
                  onClick={() => onReplaceNode(bestReplacement.type)}
                >
                  {t("workbench.studio.replaceWithCandidate", {
                    defaultValue: "Replace with {{candidate}}",
                    candidate: bestReplacement.name,
                  })}
                </button>
              ) : null}
            </div>
          ))
        )}
      </div>
      <div className="studio-inspector-section">
        <div className="studio-score">
          <span>
            <b>{checks.passed}</b>
            <small>
              {t("workbench.studio.checksPassing", {
                defaultValue: "{{passed}} of {{total}} checks pass",
                passed: checks.passed,
                total: checks.total,
              })}
            </small>
          </span>
          <span className="studio-score-bar" aria-hidden>
            <i style={{ width: `${Math.round((checks.passed / checks.total) * 100)}%` }} />
          </span>
        </div>
      </div>
      <div className="studio-inspector-section">
        <span className="wb-caption">
          {t("workbench.studio.layoutChecks", { defaultValue: "Layout" })}
        </span>
        <VerdictRow
          verdict={overflowVerdict}
          label={t("workbench.studio.checkOverflow", { defaultValue: "No horizontal overflow" })}
          detail={
            overflowFrames.length === 0
              ? t("workbench.studio.checkOverflowPass", {
                  defaultValue: "Measured clean at {{count}} frames",
                  count: measuredFrameCount,
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
          label={t("workbench.studio.checkStates", { defaultValue: "States are declared" })}
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
  );
}
