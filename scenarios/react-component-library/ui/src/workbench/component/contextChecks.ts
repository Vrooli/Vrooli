import type { Verdict } from "../verdicts";
import { overflowAt } from "./instruments";
import type { SlotBox, SlotRoleMatch } from "./placement";
import { REFERENCE_WIDTHS } from "./viewports";

/**
 * The checks the in-context view reports. They are built from measurements,
 * never from the template's promises: an instrument that has not run, or whose
 * reading is missing or implausible, leaves its row unmeasured rather than
 * showing a pass.
 */

/** Component fit is separate from the page's own health. */
export type ContextCheckGroup = "fit" | "page" | "evidence" | "human" | "info";

export interface ContextCheckRow {
  id: string;
  label: string;
  detail: string;
  verdict: Verdict;
  /** Squint is a human instrument and is excluded from the pass ratio. */
  scored: boolean;
  /** Assigned once at the end of building; absent means component fit. */
  group?: ContextCheckGroup;
}

const GROUP_BY_ID: Record<string, ContextCheckGroup> = {
  slot: "fit",
  touch: "fit",
  contrast: "fit",
  overflow: "page",
  type: "info",
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
  type: { fontFamily: string; fontSize: string } | null;
  touch: { on: boolean; small: number; total: number } | null;
  contrast: {
    on: boolean;
    ratio: number | null;
    label: string;
    background: string;
  };
  states: {
    total: number;
    covered: number;
    status: string;
    captures?: number;
    latest?: string;
    message?: string;
  } | null;
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

const UNKNOWN_SIZE = "—";

function shortFontFamily(value: string): string {
  return value.split(",")[0]?.replace(/["']/g, "").trim() || value;
}

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

export interface TouchReading {
  /** False when the instrument is off or the frame has not reported yet. */
  on: boolean;
  total: number;
  small: number;
  /** The frame is still painting, so the absence of a scan is not a finding. */
  pending?: boolean;
}

/** One touch-target row for both tabs. */
export function buildTouchRow(t: Translate, reading: TouchReading): ContextCheckRow {
  const plausible =
    reading.on &&
    reading.total > 0 &&
    Number.isFinite(reading.small) &&
    reading.small >= 0 &&
    reading.small <= reading.total;
  return {
    id: "touch",
    label: t("workbench.component.ctxCheckTouch", { defaultValue: "Touch targets ≥ 44 px" }),
    detail: !reading.on
      ? reading.pending
        ? t("workbench.component.fixtureCheckMeasuring", { defaultValue: "Measuring this frame…" })
        : t("workbench.component.ctxCheckTouchOff", {
            defaultValue: "Turn on Targets to outline every interactive element.",
          })
      : reading.total === 0
        ? t("workbench.component.ctxCheckTouchNone", {
            defaultValue: "No interactive elements.",
          })
        : !plausible
          ? t("workbench.component.ctxCheckTouchInvalid", {
              defaultValue: "The target scan was inconsistent, so it was not scored.",
            })
          : t("workbench.component.ctxCheckTouchDetail", {
              defaultValue: "{{small}} of {{total}} interactive elements are under 44 px",
              small: reading.small,
              total: reading.total,
            }),
    verdict: !plausible ? "unmeasured" : reading.small > 0 ? "warn" : "pass",
    scored: true,
    group: "fit",
  };
}

export interface ContrastReading {
  /** False when the instrument is off or the frame has not reported yet. */
  on: boolean;
  ratio: number | null;
  label: string;
  /** Empty or absent means the background could not be resolved. */
  background?: string;
  /** There is no readable text to measure; that is not a failed contrast. */
  noText?: boolean;
  pending?: boolean;
}

/** One contrast row for both tabs. */
export function buildContrastRow(t: Translate, reading: ContrastReading): ContextCheckRow {
  const ratio = isRatioPlausible(reading.ratio) ? reading.ratio : null;
  const backgroundKnown = reading.background === undefined || reading.background.length > 0;
  let verdict: Verdict = "unmeasured";
  if (reading.on && ratio !== null && backgroundKnown) {
    verdict = ratio >= 4.5 ? "pass" : ratio >= 3 ? "warn" : "fail";
  }
  return {
    id: "contrast",
    label: t("workbench.component.ctxCheckContrast", { defaultValue: "Text contrast" }),
    detail: !reading.on
      ? reading.pending
        ? t("workbench.component.fixtureCheckMeasuring", { defaultValue: "Measuring this frame…" })
        : t("workbench.component.ctxCheckContrastOff", {
            defaultValue: "Turn on Contrast, then pick a text node.",
          })
      : reading.noText
        ? t("workbench.component.ctxCheckContrastNoText", {
            defaultValue:
              "No readable text in this story, so there is no text contrast to measure.",
          })
        : ratio === null
          ? t("workbench.component.ctxCheckContrastPick", {
              defaultValue: "Pick a text node inside the frame.",
            })
          : t("workbench.component.ctxCheckContrastDetail", {
              defaultValue: "{{ratio}}:1 · {{label}}",
              ratio: ratio.toFixed(1),
              label: reading.label,
            }),
    verdict,
    scored: true,
    group: "fit",
  };
}

export interface TypeReading {
  measured: boolean;
  typography: { fontFamily: string; fontSize: string } | null;
  pending?: boolean;
}

/** Type is context, not a defect, so it is always an informational footnote. */
export function buildTypeNote(t: Translate, reading: TypeReading): ContextCheckRow {
  return {
    id: "type",
    label: t("workbench.component.ctxCheckType", { defaultValue: "Inherits page type" }),
    detail: !reading.measured
      ? reading.pending
        ? t("workbench.component.fixtureCheckMeasuring", { defaultValue: "Measuring this frame…" })
        : t("workbench.component.ctxCheckTypePending", {
            defaultValue: "Reading the slot's computed styles…",
          })
      : !reading.typography
        ? t("workbench.component.ctxCheckTypeNone", {
            defaultValue: "Type: no readable text in this story.",
          })
        : t("workbench.component.ctxCheckTypeNote", {
            defaultValue: "Type: {{font}} {{size}} · informational.",
            font: shortFontFamily(reading.typography.fontFamily),
            size: reading.typography.fontSize,
          }),
    verdict: "unmeasured",
    scored: false,
    group: "info",
  };
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
            defaultValue: "{{role}} slot names {{kind}} · {{size}} fits {{slotWidth}} px",
            role: facts.slot.role ?? openRole,
            kind: facts.component.kind,
            size,
            slotWidth: facts.slot.box.w,
          })
        : facts.slot.roleMatch === "accepted"
          ? t("workbench.component.ctxCheckSlotAccepted", {
              defaultValue: "Slot role is {{role}}; it accepts {{kind}} by kind.",
              role: facts.slot.role ?? openRole,
              kind: facts.component.kind,
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
      verdict: !declared
        ? "warn"
        : facts.slot.compatible &&
            (facts.slot.roleMatch === "exact" || facts.slot.roleMatch === "family")
          ? "pass"
          : "warn",
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
      measured.length < REFERENCE_WIDTHS.length
        ? "unmeasured"
        : offenders.length > 0
          ? "fail"
          : "pass",
    scored: true,
  });

  // 3 · type is context, not a defect. The reading must carry a real family and
  // size; the placeholder dash means the styles were not readable.
  const typeReadable =
    facts.type !== null &&
    facts.type.fontFamily.length > 0 &&
    facts.type.fontFamily !== UNKNOWN_SIZE &&
    facts.type.fontSize.length > 0 &&
    facts.type.fontSize !== UNKNOWN_SIZE;
  rows.push(
    buildTypeNote(t, {
      measured: facts.type !== null,
      typography: typeReadable && facts.type ? facts.type : null,
    }),
  );

  // 4 · touch targets. A scan that found no interactive element proves
  // nothing, and a count larger than its own total is not a measurement.
  rows.push(buildTouchRow(t, facts.touch ?? { on: false, total: 0, small: 0 }));

  // 5 · contrast probe. A ratio is only a measurement when both colours were
  // readable and the result is inside the WCAG range.
  rows.push(
    buildContrastRow(t, {
      on: facts.contrast.on,
      ratio: facts.contrast.ratio,
      label: facts.contrast.label,
      background: facts.contrast.background,
      noText: facts.type !== null && !typeReadable,
    }),
  );

  // 6 · state coverage from experience evidence.
  const statesPlausible =
    facts.states !== null &&
    facts.states.total > 0 &&
    Number.isFinite(facts.states.covered) &&
    facts.states.covered >= 0 &&
    facts.states.covered <= facts.states.total &&
    facts.states.covered > 0;
  const captureNote =
    facts.states?.captures && facts.states.captures > 0
      ? t("workbench.component.ctxCheckStatesCaptures", {
          defaultValue: "Capture records: {{count}}{{latest}}",
          count: facts.states.captures,
          latest: facts.states.latest
            ? t("workbench.component.ctxCheckStatesLatest", {
                defaultValue: " · last {{date}}",
                date: facts.states.latest.slice(0, 10),
              })
            : "",
        })
      : "";
  const captureMessage = (facts.states?.message ?? "").trim();
  if (!facts.states || facts.states.total === 0) {
    rows.push({
      id: "states",
      label: t("workbench.component.ctxCheckStates", {
        defaultValue: "Declared states are covered",
      }),
      detail: t("workbench.component.ctxCheckStatesNone", {
        defaultValue: "No experience contract declares states for this asset.",
      }),
      verdict: "unmeasured",
      scored: true,
    });
  } else if (!statesPlausible) {
    rows.push({
      id: "states",
      label:
        facts.states.status === "unavailable"
          ? t("workbench.component.ctxCheckStatesUnavailable", {
              defaultValue: "Capture unavailable",
            })
          : t("workbench.component.ctxCheckStatesNoPass", {
              defaultValue: "No passing capture",
            }),
      detail: captureNote
        ? captureMessage
          ? `${captureNote} · ${captureMessage}`
          : captureNote
        : t("workbench.component.ctxCheckStatesNoEvidence", {
            defaultValue: "The contract declares {{total}} states but no capture names them.",
            total: facts.states.total,
          }),
      verdict: "unmeasured",
      scored: true,
    });
  } else {
    rows.push({
      id: "states",
      label: t("workbench.component.ctxCheckStates", {
        defaultValue: "Declared states are covered",
      }),
      detail: t("workbench.component.ctxCheckStatesDetail", {
        defaultValue: "{{covered}}/{{total}} declared states have a passing capture.",
        covered: facts.states.covered,
        total: facts.states.total,
      }),
      // Complete coverage is a pass. Freshness is a separate question, and the
      // old "fresh" branch could never run because no producer emitted it.
      verdict: facts.states.covered === facts.states.total ? "pass" : "warn",
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
  const safeOk = safe !== null && safe.top >= safe.safeTop - 1 && safe.gap >= safe.safeBottom - 1;
  rows.push({
    id: "safeArea",
    label: t("workbench.component.ctxCheckSafeArea", {
      defaultValue: "Content respects the safe area",
    }),
    detail:
      !safe || !safe.on
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
          defaultValue:
            "Turn on Blur to check hierarchy at a distance. Nothing is measured or scored.",
        }),
    verdict: "unmeasured",
    scored: false,
  });

  for (const row of rows) row.group = GROUP_BY_ID[row.id] ?? "fit";
  return rows;
}

export interface CheckSummary {
  pass: number;
  warn: number;
  fail: number;
  unmeasured: number;
  /** Page rows that need attention. Zero for the component-fit summary. */
  pageIssues: number;
}

/** Counts one scope. Info and human rows are never part of the fit summary. */
export function summarizeCheckRows(
  rows: ContextCheckRow[],
  group: ContextCheckGroup = "fit",
): CheckSummary {
  const scoped = rows.filter((row) => (row.group ?? "fit") === group && row.scored);
  return {
    pass: scoped.filter((row) => row.verdict === "pass").length,
    warn: scoped.filter((row) => row.verdict === "warn").length,
    fail: scoped.filter((row) => row.verdict === "fail").length,
    unmeasured: scoped.filter((row) => row.verdict === "unmeasured").length,
    pageIssues:
      group === "page"
        ? scoped.filter((row) => row.verdict === "fail" || row.verdict === "warn").length
        : 0,
  };
}
