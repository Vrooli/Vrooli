import { useTranslation } from "../../i18n";
import { Chip } from "../primitives";
import type { PlacementDecision } from "./placement";

interface InContextPlacementPanelProps {
  decision: PlacementDecision;
  slotHeight: number;
  onSlotChoice: (slotId: string | null) => void;
}

export function InContextPlacementPanel({
  decision,
  slotHeight,
  onSlotChoice,
}: InContextPlacementPanelProps) {
  const { t } = useTranslation();
  const best = decision.auto;
  const chosen = decision.chosen;
  const why =
    decision.manual && chosen
      ? chosen.slotId === best?.slotId
        ? t("workbench.component.ctxWhyPinned", {
            defaultValue: "{{slot}} is pinned by hand; it is also the best score at {{score}}%.",
            slot: chosen.label,
            score: chosen.score,
          })
        : t("workbench.component.ctxWhyManual", {
            defaultValue: "{{slot}} was chosen by hand; {{auto}} still scores highest at {{score}}%.",
            slot: chosen.label,
            auto: best?.label ?? "—",
            score: best?.score ?? 0,
          })
      : chosen
        ? t("workbench.component.ctxWhy", {
            defaultValue:
              "{{slot}} wins for a {{kind}}: role {{role}}, {{fit}} of {{width}} px available, slot {{index}} on the page.",
            slot: chosen.label,
            kind: decision.profile.kind,
            role: chosen.role ?? t("workbench.component.ctxRoleOpen", { defaultValue: "open" }),
            fit: `${decision.profile.size.w} px`,
            width: chosen.box.w,
            index: chosen.index + 1,
          })
        : t("workbench.component.ctxWhyEmpty", {
            defaultValue: "This page declares no slots, so the component cannot be placed.",
          });

  return (
    <div
      className="ic-side"
      role="group"
      aria-label={t("workbench.component.placement", { defaultValue: "Placement" })}
    >
      <span className="ic-eyebrow">
        {t("workbench.component.placement", { defaultValue: "Placement" })}
      </span>
      {chosen ? (
        <>
          <div className="ic-chosen">
            <div className="ic-chosen-head">
              <b>{chosen.label}</b>
              <Chip tone={chosen.compatible ? "green" : "amber"}>{chosen.score}%</Chip>
            </div>
            <p className="ic-why">{why}</p>
            <p className="ic-meta">
              {t("workbench.component.ctxSlotMeta", {
                defaultValue: "role {{role}} · {{width}} × {{height}} · accepts {{accepts}}",
                role: chosen.role ?? t("workbench.component.ctxRoleOpen", { defaultValue: "open" }),
                width: chosen.box.w,
                height: Math.round(slotHeight),
                accepts: chosen.accepts.length
                  ? chosen.accepts.join(", ")
                  : t("workbench.component.ctxAcceptsAny", { defaultValue: "any kind" }),
              })}
            </p>
            <p className="ic-flow">
              {t("workbench.component.ctxFlow", {
                defaultValue:
                  "In flow: the page lays the specimen out inside {{slot}}, so the slot sets its width. Pin another slot below to compare.",
                slot: chosen.label,
              })}
            </p>
          </div>

          {!chosen.compatible ? (
            <p className="ic-note is-warn" role="status">
              {t("workbench.component.ctxMismatch", {
                defaultValue:
                  "No slot on this page declares {{kind}}. It is placed unproven — pick another slot or a different page template.",
                kind: decision.profile.kind,
              })}
            </p>
          ) : null}

          <div
            className="ic-slots"
            role="group"
            aria-label={t("workbench.component.slotPicker", { defaultValue: "Slot picker" })}
          >
            {decision.candidates.map((candidate) => (
              <button
                key={candidate.slotId}
                type="button"
                className="ic-slot-row"
                aria-pressed={candidate.slotId === chosen.slotId}
                onClick={() => onSlotChoice(candidate.slotId)}
              >
                <span
                  className={`ic-slot-mark${candidate.compatible ? "" : " is-mismatch"}`}
                  aria-hidden
                />
                <span className="ic-slot-label">
                  <span className="ic-slot-name">{candidate.label}</span>
                  {candidate.slotId === decision.auto?.slotId ? (
                    <span className="ic-slot-auto">
                      {t("workbench.component.ctxAuto", { defaultValue: "auto" })}
                    </span>
                  ) : null}
                  <span className="ic-slot-kind">
                    {candidate.role ??
                      t("workbench.component.ctxRoleOpen", { defaultValue: "open" })}
                  </span>
                </span>
                <span className="ic-slot-score">{candidate.score}</span>
                <span className="ic-affinity" aria-hidden>
                  <i style={{ width: `${candidate.score}%` }} />
                </span>
              </button>
            ))}
          </div>

          {decision.manual ? (
            <button type="button" className="ic-reset" onClick={() => onSlotChoice(null)}>
              {t("workbench.component.resetPlacement", {
                defaultValue: "Reset to authored slot",
              })}
            </button>
          ) : null}
        </>
      ) : (
        <div className="ic-note is-warn" role="status">
          <b>{t("workbench.component.ctxNoSlotTitle", { defaultValue: "No slot on this page" })}</b>
          <p>
            {t("workbench.component.ctxNoSlot", {
              defaultValue:
                "This page has no component slot to place into. Choose a page template that declares one, or add a slot in the studio.",
            })}
          </p>
        </div>
      )}
    </div>
  );
}
