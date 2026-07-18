/** The persisted initiative strategy value for per-item workflows. */
export const MEMBER_ITEM_STRATEGY_WIRE_VALUE = "item-level";

/** Operator-facing label for the only supported initiative strategy. */
export const MEMBER_ITEM_STRATEGY_LABEL = "Member-item workflow";

export type ModePresentation =
  | { kind: "member-item-strategy"; wireValue: typeof MEMBER_ITEM_STRATEGY_WIRE_VALUE; label: typeof MEMBER_ITEM_STRATEGY_LABEL }
  | { kind: "legacy"; mode: string };

export function resolveModePresentation(mode: string | null | undefined): ModePresentation {
  const normalized = (mode ?? "").trim();
  if (normalized === "" || normalized === MEMBER_ITEM_STRATEGY_WIRE_VALUE) {
    return { kind: "member-item-strategy", wireValue: MEMBER_ITEM_STRATEGY_WIRE_VALUE, label: MEMBER_ITEM_STRATEGY_LABEL };
  }
  return { kind: "legacy", mode: normalized };
}

export function isMemberItemStrategy(mode: string | null | undefined): boolean {
  return resolveModePresentation(mode).kind === "member-item-strategy";
}

export function normalizeModeWireValue(mode: string | null | undefined): string {
  const presentation = resolveModePresentation(mode);
  return presentation.kind === "member-item-strategy" ? presentation.wireValue : presentation.mode;
}

export function presentModeLabel(mode: string | null | undefined, legacyLabel?: string): string {
  const presentation = resolveModePresentation(mode);
  return presentation.kind === "member-item-strategy" ? presentation.label : legacyLabel?.trim() || presentation.mode;
}
