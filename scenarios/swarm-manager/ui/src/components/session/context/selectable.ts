/**
 * Shared selection-mode contract for context-entity cards.
 *
 * One card component per entity type is consumed by BOTH the sidebar (its
 * native navigate/action behavior) and the SessionContextPicker drawer. When
 * a card receives a `CardSelection` with `selectionMode: true` it renders in
 * "pick mode": a single toggle affordance with a checkbox + selected ring,
 * no navigation, no action buttons, and a disabled state when a context cap
 * is reached.
 *
 * Selection POLICY (caps, which items are disabled) lives in the picker —
 * cards never read CONTEXT_TYPE_CAPS. They only render the computed state.
 */
export interface CardSelection {
  /** When true, render the card in picker pick-mode (checkbox + toggle, no actions). */
  selectionMode: boolean;
  /** Whether this entity is currently selected. */
  selected: boolean;
  /** Cap reached and this entity is not already selected — selection is refused. */
  disabled?: boolean;
  /** Human-readable reason shown when `disabled` (e.g. the cap message). */
  disabledReason?: string;
  /** Toggle this entity's selection. No-op while `disabled`. */
  onToggleSelect?: () => void;
}
