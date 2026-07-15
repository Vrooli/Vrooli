/**
 * Policy-controls labeling for settings sections whose values are consumed by
 * the operation runner's transition policies (via the PolicyControls seam)
 * rather than being pure UI preferences. Grouping + labeling only — the
 * controls themselves stay editable exactly as before.
 */

import type { SettingsPolicyProjection } from "../../types";

export interface PolicyControlsNoteProps {
  /** Persisted settings field names (JSON names) covered by this section. */
  fields: string[];
  /** Optional projection from the API; enables per-field destination labels. */
  projection?: SettingsPolicyProjection | null;
}

/** Small badge marking a settings card as policy-level. */
export function PolicyControlsBadge() {
  return (
    <span
      className="rounded-full border border-indigo-400/30 bg-indigo-500/10 px-2 py-0.5 text-[11px] font-medium text-indigo-300"
      data-testid="policy-controls-badge"
      title="These settings govern the operation runner's transition policies (auto-advance, retries, review spawning). They remain user preferences; orchestration reads them through the policy-controls seam."
    >
      Policy controls
    </span>
  );
}

/**
 * Explanatory note under a policy-level section. When the API projection is
 * available, lists each field's destination control path.
 */
export function PolicyControlsNote({ fields, projection }: PolicyControlsNoteProps) {
  const classified = projection
    ? projection.classifications.filter((c) => fields.includes(c.field) && c.control)
    : [];
  return (
    <div className="mt-2 rounded border border-indigo-400/20 bg-indigo-500/5 px-3 py-2" data-testid="policy-controls-note">
      <p className="text-xs text-indigo-200/80">
        These settings govern the operation runner&apos;s transition policies. Your
        preferences are kept; the runner reads them through typed policy controls
        instead of raw settings.
      </p>
      {classified.length > 0 && (
        <ul className="mt-1.5 space-y-0.5">
          {classified.map((c) => (
            <li key={c.field} className="text-[11px] text-slate-500">
              <code className="text-slate-400">{c.field}</code>
              {" → "}
              <code className="text-indigo-300/80">{c.control}</code>
              {c.role === "dormant" ? <span className="ml-1 text-amber-400/80">(dormant — no runtime reader)</span> : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
