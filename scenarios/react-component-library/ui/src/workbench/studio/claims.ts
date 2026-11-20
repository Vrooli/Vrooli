import type { TFunction } from "i18next";

import type { SlotPlacement, TemplateNode } from "../templates/spec";
import { walk } from "../templates/spec";
import { captureEvidenceVerdict, primaryActionVerdict } from "./checks";
import type { StudioClaim } from "./StudioInspector";

export interface ClaimContext {
  /** A live component placed in the composition, when one is attached. */
  componentId?: string;
  /** Where that component sits inside the frame. */
  placement?: SlotPlacement;
}

/**
 * The studio's standing claims about one frame. Both studio surfaces share this
 * implementation so a composition is judged by one vocabulary, not two.
 */
export function claimsFor(spec: TemplateNode, t: TFunction, context: ClaimContext = {}): StudioClaim[] {
  let primary = 0;
  let displays = 0;
  let titles = 0;
  let bodies = 0;
  let slots = 0;
  walk(spec, (current) => {
    if (current.type === "button" && current.props.variant === "primary") primary += 1;
    if (current.type === "slot") slots += 1;
    if (current.type === "text" && current.props.variant === "display") displays += 1;
    if (current.type === "text" && current.props.variant === "title") titles += 1;
    if (current.type === "text" && current.props.variant === "body") bodies += 1;
  });
  return [
    {
      id: "slot",
      label: t("workbench.studio.checkSlot", {
        defaultValue: "Composition declares a component slot",
      }),
      verdict: slots > 0 ? "pass" : "warn",
      detail: context.componentId
        ? context.placement
          ? t("workbench.studio.slotPlaced", {
              defaultValue: "Live component placed · {{width}} × {{height}}",
              width: Math.round(context.placement.w),
              height: Math.round(context.placement.h),
            })
          : t("workbench.studio.slotInFlow", {
              defaultValue: "Live component sits in the designed slot",
            })
        : t("workbench.studio.slotCount", { count: slots }),
    },
    {
      id: "primary",
      label: t("workbench.studio.checkPrimary", { defaultValue: "A primary action is declared" }),
      verdict: primaryActionVerdict(primary),
      detail: t("workbench.studio.primaryCount", {
        defaultValue: "{{buttons}} primary buttons · {{slots}} component slots",
        buttons: primary,
        slots,
      }),
    },
    {
      id: "hierarchy",
      label: t("workbench.studio.checkHierarchy", {
        defaultValue: "Display, title and body hierarchy exists",
      }),
      verdict: displays > 0 && titles > 0 && bodies > 0 ? "pass" : "warn",
      detail: t("workbench.studio.hierarchyCount", {
        defaultValue: "{{display}} display · {{title}} title · {{body}} body",
        display: displays,
        title: titles,
        body: bodies,
      }),
    },
    {
      id: "evidence",
      label: t("workbench.studio.checkEvidence", {
        defaultValue: "Capture evidence is attached",
      }),
      verdict: captureEvidenceVerdict(false),
      detail: context.componentId
        ? t("workbench.studio.evidenceLiveOnly", {
            defaultValue: "Live component attached · no capture for this revision",
          })
        : t("workbench.studio.evidenceNone", {
            defaultValue: "No captured evidence for this revision",
          }),
    },
    {
      id: "contrast",
      label: t("workbench.studio.checkContrast", { defaultValue: "Contrast is measured" }),
      verdict: "unmeasured",
      detail: t("workbench.studio.contrastNote", {
        defaultValue: "Not captured for this local composition",
      }),
    },
  ];
}
