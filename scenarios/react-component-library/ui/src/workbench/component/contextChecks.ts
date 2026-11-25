import { overflowAt } from "./instruments";
import type { SlotBox, SlotRoleMatch } from "./placement";
import { REFERENCE_WIDTHS } from "./viewports";

/**
 * The checks the in-context view reports. They are built from measurements,
 * never from the template's promises: an instrument that has not run, or whose
 * reading is missing or implausible, leaves its row unmeasured rather than
 * showing a pass.
 */

export type ContextVerdict = "pass" | "fail" | "warn" | "unmeasured";

export type CheckEvidence = "measured" | "declared" | "none";

/** Component fit is separate from the page's own health. */
export type ContextCheckGroup = "fit" | "page" | "evidence" | "human";

export interface ContextCheckRow {
  id: string;
  label: string;
  detail: string;
  verdict: ContextVerdict;
  evidence: CheckEvidence;
  /** Squint is a human instrument and is excluded from the pass ratio. */
  scored: boolean;
  group?: ContextCheckGroup;
}

const GROUP_BY_ID: Record<string, ContextCheckGroup> = {
  slot: "fit",
  touch: "fit",
  contrast: "fit",
  overflow: "page",
  type: "page",
  safeArea: "page",
  states: "evidence",
  squint: "human",
};

export interface ContextFacts {
  slot:
    | {
        label: string;
        /** Absent when the template authors a slot without a role. */
        role: string | undefined;
        /** Empty when the template declares no accepted kinds. */
        accepts: string[];
        compatible: boolean;
        roleMatch: SlotRoleMatch;
        box: SlotBox;
      }
    | undefined;
  component: { kind: string; size: SlotBox };
  /**
   * One row per measured page-only probe. A pass needs all three reference
   * widths and their measured `scrollWidth`; a row without it was not measured.
   */
  overflow: Array<{ width: number; overflow: boolean; scrollWidth?: number }>;
  /** Null until the live slot's computed styles have been read. */
  type: { inherited: boolean; fontFamily: string; fontSize: string } | null;
  touch: { on: boolean; small: number; total: number } | null;
  contrast: {
    on: boolean;
    ratio: number | null;
    label: string;
    foreground: string;
    background: string;
  };
  states: { total: number; covered: number; status: string } | null;
  safeArea: {
    on: boolean;
    /** Offset of the first page content from the frame top. */
    top: number;
    /** Inset the frame keeps above the home indicator / edge. */
    gap: number;
    height: number;
    safeTop: number;
    safeBottom: number;
  } | null;
  squint: { on: boolean };
}

export type Translate = (key: string, options?: Record<string, unknown>) => string;

const UNKNOWN_ROLE = "—";
const UNKNOWN_SIZE = "—";

function isRatioPlausible(ratio: number | null): ratio is number {
  return ratio !== null && Number.isFinite(ratio) && ratio >= 1 && ratio <= 21;
}

function isMeasurableOverflow(row: { width: number; scrollWidth?: number }): boolean {
  return (
    Number.isFinite(row.width) &&
    row.width > 0 &&
    row.scrollWidth !== undefined &&
    Number.isFinite(row.scrollWidth) &&
    // A zero scrollWidth means the probe never laid out; that is not a
    // measurement that can prove the absence of overflow.
    row.scrollWidth > 0
  );
}

/** The overflow a row proves. A supplied flag never overrides a measurement. */
function overflowOf(row: { width: number; overflow: boolean; scrollWidth?: number }): boolean {
  if (row.scrollWidth === undefined) return row.overflow;
  return overflowAt(row.width, row.scrollWidth);
}

export function buildContextChecks(facts: ContextFacts, t: Translate): ContextCheckRow[] {
  const rows: ContextCheckRow[] = [];
  const openRole = t("workbench.component.ctxRoleOpen", { defaultValue: "open" });
  const anyKind = t("workbench.component.ctxCheckSlotOpen", { defaultValue: "anything" });

  // 1 · slot accepts the component. A slot only proves compatibility when it
  // declares both the role it plays and the kinds it accepts; an untyped slot
  // ("open") is usable but unproven, so it warns instead of passing.
  if (!facts.slot) {
    rows.push({
      id: "slot",
      label: t("workbench.component.ctxCheckSlot", { defaultValue: "Slot accepts this component" }),
      detail: t("workbench.component.ctxCheckSlotMissing", {
        defaultValue: "This page declares no component slot, so nothing was placed.",
      }),
      verdict: "fail",
      evidence: "declared",
      scored: true,
    });
  } else {
    const declared = Boolean(facts.slot.role) && facts.slot.accepts.length > 0;
    const size = `${facts.component.size.w} × ${facts.component.size.h}`;
    const detail = !declared
      ? t("workbench.component.ctxCheckSlotUndeclared", {
          defaultValue:
            "{{role}} slot declares {{accepts}}; add a role and accepted kinds to prove the match.",
          role: facts.slot.role ?? openRole,
          accepts: facts.slot.accepts.join(", ") || anyKind,
        })
      : facts.slot.roleMatch === "exact"
        ? t("workbench.component.ctxCheckSlotOk", {
            defaultValue:
              "{{role}} slot names {{kind}} · {{size}} fits {{slotWidth}} px",
            role: facts.slot.role ?? openRole,
            kind: facts.component.kind,
            size,
            slotWidth: facts.slot.box.w,
          })
        : facts.slot.roleMatch === "accepted"
          ? t("workbench.component.ctxCheckSlotAccepted", {
              defaultValue:
                "{{role}} slot accepts {{kind}} by kind · {{size}} fits {{slotWidth}} px, but the role is not exact.",
              role: facts.slot.role ?? openRole,
              kind: facts.component.kind,
              size,
              slotWidth: facts.slot.box.w,
            })
          : facts.slot.roleMatch === "family"
            ? t("workbench.component.ctxCheckSlotFamily", {
                defaultValue:
                  "{{role}} is a related region for {{kind}} · {{size}} fits {{slotWidth}} px.",
                role: facts.slot.role ?? openRole,
                kind: facts.component.kind,
                size,
                slotWidth: facts.slot.box.w,
              })
            : facts.slot.roleMatch === "role-only"
              ? t("workbench.component.ctxCheckSlotRoleOnly", {
                  defaultValue:
                    "{{role}} slot declares no accepted kinds · {{size}} fits {{slotWidth}} px, so the match is unproven.",
                  role: facts.slot.role ?? openRole,
                  size,
                  slotWidth: facts.slot.box.w,
                })
              : t("workbench.component.ctxCheckSlotMismatch", {
                  defaultValue:
                    "{{role}} slot accepts {{accepts}}; {{kind}} is not declared compatible",
                  role: facts.slot.role ?? openRole,
                  accepts: facts.slot.accepts.join(", ") || anyKind,
                  kind: facts.component.kind,
                });
    rows.push({
      id: "slot",
      label: t("workbench.component.ctxCheckSlot", { defaultValue: "Slot accepts this component" }),
      detail,
      verdict:
        !declared
          ? "warn"
          : facts.slot.compatible &&
              (facts.slot.roleMatch === "exact" || facts.slot.roleMatch === "family")
            ? "pass"
            : "warn",
      evidence: "declared",
      scored: true,
    });
  }

  // 2 · no horizontal overflow, measured at the three reference widths. Every
  // width needs a real `scrollWidth`; a partial or unreadable scan is
  // unmeasured, never a pass.
  const measured = REFERENCE_WIDTHS.map((width) =>
    facts.overflow.find((row) => row.width === width),
  ).filter(
    (row): row is { width: number; overflow: boolean; scrollWidth?: number } =>
      Boolean(row) && isMeasurableOverflow(row as { width: number; scrollWidth?: number }),
  );
  const offenders = measured.filter((row) => overflowOf(row));
  const overflowEvidence = measured.length === 0 ? "none" : "measured";
  rows.push({
    id: "overflow",
    label: t("workbench.component.ctxCheckOverflow", { defaultValue: "No horizontal overflow" }),
    detail:
      measured.length < REFERENCE_WIDTHS.length
        ? t("workbench.component.ctxCheckOverflowPending", {
            defaultValue: "Measuring {{widths}}…",
            widths: REFERENCE_WIDTHS.map((width) => `${width} px`).join(", "),
          })
        : t("workbench.component.ctxCheckOverflowDetail", {
            defaultValue: "Measured at {{widths}}",
            widths: measured
              .map((row) => `${row.width} ${overflowOf(row) ? "✗" : "✓"}`)
              .join(" · "),
          }),
    verdict:
      measured.length < REFERENCE_WIDTHS.length ? "unmeasured" : offenders.length > 0 ? "fail" : "pass",
    evidence: overflowEvidence,
    scored: true,
  });

  // 3 · inherits page type. The reading must carry a real family and size; the
  // placeholder dash means the styles were not readable.
  const typeReadable =
    facts.type !== null &&
    facts.type.fontFamily.length > 0 &&
    facts.type.fontFamily !== UNKNOWN_SIZE &&
    facts.type.fontSize.length > 0 &&
    facts.type.fontSize !== UNKNOWN_SIZE;
  rows.push({
    id: "type",
    label: t("workbench.component.ctxCheckType", { defaultValue: "Inherits page type" }),
    detail: typeReadable
      ? t("workbench.component.ctxCheckTypeDetail", {
          defaultValue: "Slot computes {{font}} at {{size}} inside the page",
          font: facts.type?.fontFamily ?? UNKNOWN_ROLE,
          size: facts.type?.fontSize ?? UNKNOWN_SIZE,
        })
      : t("workbench.component.ctxCheckTypePending", {
          defaultValue: "Reading the slot's computed styles…",
        }),
    verdict: !typeReadable ? "unmeasured" : facts.type?.inherited ? "pass" : "warn",
    evidence: typeReadable ? "measured" : "none",
    scored: true,
  });

  // 4 · touch targets, measured by the Targets instrument. A scan that found no
  // interactive element proves nothing, and a count larger than its own total
  // is not a measurement.
  const touchSmall = facts.touch?.small ?? 0;
  const touchPlausible =
    facts.touch !== null &&
    facts.touch.on &&
    facts.touch.total > 0 &&
    Number.isFinite(facts.touch.small) &&
    facts.touch.small >= 0 &&
    facts.touch.small <= facts.touch.total;
  rows.push({
    id: "touch",
    label: t("workbench.component.ctxCheckTouch", { defaultValue: "Touch targets ≥ 44 px" }),
    detail: !facts.touch || !facts.touch.on
      ? t("workbench.component.ctxCheckTouchOff", {
          defaultValue: "Turn on Targets to outline every interactive element.",
        })
      : facts.touch.total === 0
        ? t("workbench.component.ctxCheckTouchNone", {
            defaultValue: "No interactive elements were found, so nothing was measured.",
          })
        : !touchPlausible
          ? t("workbench.component.ctxCheckTouchInvalid", {
              defaultValue: "The target scan was inconsistent, so it was not scored.",
            })
          : t("workbench.component.ctxCheckTouchDetail", {
              defaultValue: "{{small}} of {{total}} interactive elements are under 44 px",
              small: facts.touch.small,
              total: facts.touch.total,
            }),
    verdict: !touchPlausible ? "unmeasured" : touchSmall > 0 ? "warn" : "pass",
    evidence: touchPlausible ? "measured" : "none",
    scored: true,
  });

  // 5 · contrast probe. A ratio is only a measurement when both colours were
  // readable and the result is inside the WCAG range.
  const contrastPlausible =
    facts.contrast.on &&
    isRatioPlausible(facts.contrast.ratio) &&
    facts.contrast.background.length > 0;
  const contrastRatio = isRatioPlausible(facts.contrast.ratio) ? facts.contrast.ratio : null;
  rows.push({
    id: "contrast",
    label: t("workbench.component.ctxCheckContrast", { defaultValue: "Text contrast" }),
    detail: !facts.contrast.on
      ? t("workbench.component.ctxCheckContrastOff", {
          defaultValue: "Turn on Contrast, then pick a text node.",
        })
      : !contrastPlausible
        ? t("workbench.component.ctxCheckContrastPick", {
            defaultValue: "Pick a text node inside the frame.",
          })
        : t("workbench.component.ctxCheckContrastDetail", {
            defaultValue: "{{ratio}}:1 · {{label}} on {{background}}",
            ratio: (contrastRatio ?? 0).toFixed(1),
            label: facts.contrast.label,
            background: facts.contrast.background,
          }),
    verdict:
      !contrastPlausible || contrastRatio === null
        ? "unmeasured"
        : contrastRatio >= 4.5
          ? "pass"
          : contrastRatio >= 3
            ? "warn"
            : "fail",
    evidence: contrastPlausible ? "measured" : "none",
    scored: true,
  });

  // 6 · state coverage from experience evidence.
  const statesPlausible =
    facts.states !== null &&
    facts.states.total > 0 &&
    Number.isFinite(facts.states.covered) &&
    facts.states.covered >= 0 &&
    facts.states.covered <= facts.states.total &&
    facts.states.covered > 0;
  if (!facts.states || facts.states.total === 0) {
    rows.push({
      id: "states",
      label: t("workbench.component.ctxCheckStates", { defaultValue: "Declared states are covered" }),
      detail: t("workbench.component.ctxCheckStatesNone", {
        defaultValue: "No experience contract declares states for this asset.",
      }),
      verdict: "unmeasured",
      evidence: "none",
      scored: true,
    });
  } else if (!statesPlausible) {
    rows.push({
      id: "states",
      label: t("workbench.component.ctxCheckStates", { defaultValue: "Declared states are covered" }),
      detail: t("workbench.component.ctxCheckStatesNoEvidence", {
        defaultValue: "The contract declares {{total}} states but no capture names them.",
        total: facts.states.total,
      }),
      verdict: "unmeasured",
      evidence: "none",
      scored: true,
    });
  } else {
    rows.push({
      id: "states",
      label: t("workbench.component.ctxCheckStates", { defaultValue: "Declared states are covered" }),
      detail: t("workbench.component.ctxCheckStatesDetail", {
        defaultValue: "{{covered}}/{{total}} states have evidence · {{status}}",
        covered: facts.states.covered,
        total: facts.states.total,
        status: facts.states.status,
      }),
      verdict:
        facts.states.covered === facts.states.total && facts.states.status === "fresh"
          ? "pass"
          : "warn",
      evidence: "measured",
      scored: true,
    });
  }

  // 7 · safe area, measured by the Guides instrument. A reading with a zero
  // height or a negative offset is not a layout.
  const safe = facts.safeArea;
  const safePlausible =
    safe !== null &&
    safe.on &&
    Number.isFinite(safe.top) &&
    safe.top >= 0 &&
    safe.height > 0 &&
    Number.isFinite(safe.gap) &&
    Number.isFinite(safe.safeTop) &&
    Number.isFinite(safe.safeBottom);
  const safeOk =
    safe !== null && safe.top >= safe.safeTop - 1 && safe.gap >= safe.safeBottom - 1;
  rows.push({
    id: "safeArea",
    label: t("workbench.component.ctxCheckSafeArea", { defaultValue: "Content respects the safe area" }),
    detail: !safe || !safe.on
      ? t("workbench.component.ctxCheckSafeAreaOff", {
          defaultValue: "Turn on Guides to measure the safe area and thumb zone.",
        })
      : !safePlausible
        ? t("workbench.component.ctxCheckSafeAreaInvalid", {
            defaultValue: "The safe-area reading was incomplete, so it was not scored.",
          })
        : t("workbench.component.ctxCheckSafeAreaDetail", {
            defaultValue:
              "Content starts {{top}} px (safe {{safeTop}}) · {{gap}} px inset above the edge (safe {{safeBottom}})",
            top: Math.round(safe.top),
            safeTop: Math.round(safe.safeTop),
            gap: Math.round(safe.gap),
            safeBottom: Math.round(safe.safeBottom),
          }),
    verdict: !safePlausible ? "unmeasured" : safeOk ? "pass" : "warn",
    evidence: safePlausible ? "measured" : "none",
    scored: true,
  });

  // 8 · squint preview is human judgement, not a computed verdict. It uses the
  // same vocabulary as every other check: no capture, so unmeasured. Turning
  // blur on must never move this row out of unmeasured.
  rows.push({
    id: "squint",
    label: t("workbench.component.ctxCheckSquint", { defaultValue: "Squint preview" }),
    detail: facts.squint.on
      ? t("workbench.component.ctxCheckSquintOn", {
          defaultValue:
            "Blur is applied. Judging hierarchy by eye is the point; nothing is measured or scored.",
        })
      : t("workbench.component.ctxCheckSquintOff", {
          defaultValue: "Turn on Blur to check hierarchy at a distance. Nothing is measured or scored.",
        }),
    verdict: "unmeasured",
    evidence: "none",
    scored: false,
  });

  for (const row of rows) row.group = GROUP_BY_ID[row.id] ?? "fit";
  return rows;
}

export function scoredRows(rows: ContextCheckRow[]): ContextCheckRow[] {
  return rows.filter((row) => row.scored);
}

export function countPassed(rows: ContextCheckRow[]): number {
  return scoredRows(rows).filter((row) => row.verdict === "pass").length;
}
