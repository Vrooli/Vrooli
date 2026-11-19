import { useEffect, useState } from 'react';
import type { PolicyLever } from '../types';

interface PolicyPanelProps {
  levers: PolicyLever[];
  isSaving: boolean;
  error: string | null;
  outcomeUnknown: { key: string; requestedValue: string; currentValue: string | null; readbackError: string | null } | null;
  onSave: (key: string, value: string) => void;
  onRefresh: () => void;
  onAcknowledgeReadback: () => void;
}

const ENFORCE_OPTIONS = ['off', 'advisory', 'on'];

const POLICY_LABELS: Record<string, string> = {
  tracking_threshold: 'Minimum tracked GPU memory',
  idle_grace: 'Idle claim grace period',
  default_heartbeat_ttl: 'Default claim heartbeat lifetime',
  reconcile_warn_threshold: 'Over-claim warning threshold',
  enforce: 'Admission enforcement mode',
  preempt_enabled: 'Allow last-resort stop',
  auto_stop_allowlist: 'Owners allowed to auto-stop',
  sweep_interval: 'Claim sweep interval',
  degrade_debounce: 'Minimum time between degradation actions',
  upshift_headroom: 'Free GPU memory before restoring a workload',
  idle_yield_floor: 'Minimum priority allowed to reclaim an idle claim',
  terminal_retention: 'Keep completed claim records for',
  observed_peak_halflife: 'Observed peak memory decay time',
  default_idle_unload_ttl: 'Default idle-unload delay',
  recommend_headroom: 'Extra memory margin in right-size recommendations',
  swap_pressure_threshold: 'Swap-pressure fallback threshold',
  accel_reprobe: 'Action when accelerator readiness changes',
};

const LEVER_HELP: Record<string, string> = {
  tracking_threshold: 'Minimum observed GPU memory in bytes before reconciliation tracks a consumer; whole number, 0 or more.',
  idle_grace: 'How long a claim must remain idle before it can be reclaimed; duration such as 30s, 0 or more.',
  default_heartbeat_ttl: 'Fallback claim lifetime when an owner sends no heartbeat; duration greater than 0.',
  reconcile_warn_threshold: 'Extra GPU memory in bytes above a claim before reconciliation reports over-claim; whole number, 0 or more.',
  enforce: 'Admission behavior: off disables checks, advisory reports without blocking, on enforces admission.',
  preempt_enabled: 'Allows the final escalation step to stop a resource after degradation fails. False prevents this stop action.',
  auto_stop_allowlist: 'Comma-separated owner names eligible for automatic stop. Empty means automatic stop is off for all owners.',
  sweep_interval: 'Minimum interval between opportunistic claim sweeps; duration such as 30s, greater than 0.',
  degrade_debounce: 'Minimum time between two degradation actions for one target; duration such as 30s, 0 or more.',
  upshift_headroom: 'Free GPU memory in bytes required before a degraded idle claim may return to its preferred mode; whole number, 0 or more.',
  idle_yield_floor: 'Priority threshold for reclaiming an idle claim that opted into yielding: interactive, service, or batch.',
  terminal_retention: 'How long released, expired, or stopped claim records remain before cleanup; duration, 0 or more.',
  observed_peak_halflife: 'Time for a stored peak-memory observation to lose half its weight; duration greater than 0.',
  default_idle_unload_ttl: 'Fallback idle time before an eligible resource may unload. 0 disables this behavior; nonzero values enable it.',
  recommend_headroom: 'Extra percent above observed peak used when suggesting a smaller memory reservation; whole number, 0 or more.',
  swap_pressure_threshold: 'Swap-occupancy fallback for memory admission when fresh pressure data is unavailable; 0–100 percent, 0 disables the fallback.',
  accel_reprobe: 'When an accelerator appears: off does nothing, report records drift, restart restarts idle affected resources.',
};

const POLICY_OPTIONS: Record<string, string[]> = {
  enforce: ENFORCE_OPTIONS,
  preempt_enabled: ['false', 'true'],
  idle_yield_floor: ['interactive', 'service', 'batch'],
  accel_reprobe: ['off', 'report', 'restart'],
};

const POLICY_SCOPE = 'System Monitor capacity policy; one key is written per request.';

const POLICY_UNIT: Record<string, string> = {
  tracking_threshold: 'bytes (integer, ≥ 0)',
  idle_grace: 'duration (0 or more)',
  default_heartbeat_ttl: 'duration (greater than 0)',
  reconcile_warn_threshold: 'bytes over claim (integer, ≥ 0)',
  enforce: 'mode: off, advisory, or on',
  preempt_enabled: 'boolean',
  auto_stop_allowlist: 'comma-separated owner names',
  sweep_interval: 'duration (greater than 0)',
  degrade_debounce: 'duration (0 or more)',
  upshift_headroom: 'bytes (integer, ≥ 0)',
  idle_yield_floor: 'priority tier',
  terminal_retention: 'duration (0 or more)',
  observed_peak_halflife: 'duration (greater than 0)',
  default_idle_unload_ttl: 'duration (0 disables)',
  recommend_headroom: 'percent (integer, ≥ 0)',
  swap_pressure_threshold: 'percent (0–100; 0 disables fallback)',
  accel_reprobe: 'mode: off, report, or restart',
};

/**
 * Editable policy levers. Each row holds local draft state; Save persists one
 * lever and the parent re-fetches the full validated policy. The enforce lever
 * is a constrained select since it is an enum.
 */
export const PolicyPanel = ({ levers, isSaving, error, outcomeUnknown, onSave, onRefresh, onAcknowledgeReadback }: PolicyPanelProps) => {
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [reviewKey, setReviewKey] = useState<string | null>(null);

  useEffect(() => {
    const next: Record<string, string> = {};
    for (const lever of levers) {
      next[lever.key] = lever.value;
    }
    setDrafts(next);
  }, [levers]);

  const updateDraft = (key: string, value: string) => {
    setDrafts((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      {error && !outcomeUnknown && (
        <div role="alert" data-sm-style="sm-style-61375b27ab">
          {error}
        </div>
      )}
      {outcomeUnknown && (
        <div className="card" role="alert" aria-label="Policy update needs review">
          <h4 className="section-heading">Policy update needs review</h4>
          <p>The response for <code>{outcomeUnknown.key}</code> could not confirm whether the requested value was applied.</p>
          {error && <p>{error}</p>}
          <p>Requested value: <code>{outcomeUnknown.requestedValue}</code></p>
          {outcomeUnknown.readbackError
            ? <p>Current policy could not be confirmed: {outcomeUnknown.readbackError}</p>
            : outcomeUnknown.currentValue === null
              ? <p>The policy service did not return this setting.</p>
              : <p>Current policy readback: <code>{outcomeUnknown.currentValue}</code>. Review this value before another change.</p>}
          <div className="flex-row-center gap-sm">
            <button type="button" className="btn btn-action" onClick={onRefresh} disabled={isSaving}>
              Refresh current policy
            </button>
            <button type="button" className="btn btn-action" onClick={onAcknowledgeReadback} disabled={isSaving || Boolean(outcomeUnknown.readbackError) || outcomeUnknown.currentValue === null}>
              I reviewed the current value; allow another update
            </button>
          </div>
        </div>
      )}
      <p className="text-dim-xs">Effective values below come from the last successful policy read. Each save updates one key through the capacity policy owner; other keys are not part of that write.</p>
      <div data-sm-style="sm-style-9f304072b4">
        {levers.map((lever) => {
          const draft = drafts[lever.key] ?? lever.value;
          const dirty = draft !== lever.value;
          const inputId = `policy-${lever.key}`;
          const options = POLICY_OPTIONS[lever.key];
          return (
            <div key={lever.key} data-sm-style="sm-style-a4cd9cd628">
              <label htmlFor={inputId} className="input-label policy-lever-key" data-sm-style="sm-style-37fc90e1c1">
                {POLICY_LABELS[lever.key] ?? lever.key}
              </label>
              <div className="text-dim-xs"><code>{lever.key}</code></div>
              <div className="text-dim-xs">Effective value: <code>{lever.value}</code> · {POLICY_UNIT[lever.key] ?? 'value as accepted by the capacity policy owner'} · {POLICY_SCOPE}</div>
              <div className="policy-lever-row" data-sm-style="sm-style-070fbdbe2e">
                {options ? (
                  <select
                    id={inputId}
                    className="input-field"
                    value={draft}
                    onChange={(e) => { updateDraft(lever.key, e.target.value); }}
                    data-sm-style="sm-style-76aec44f93"
                  >
                    {options.map((opt) => (
                      <option key={opt} value={opt}>{opt}</option>
                    ))}
                  </select>
                ) : (
                  <input
                    id={inputId}
                    className="input-field policy-lever-value"
                    type="text"
                    value={draft}
                    onChange={(e) => { updateDraft(lever.key, e.target.value); }}
                    data-sm-style="sm-style-76aec44f93"
                  />
                )}
                <button
                  type="button"
                  className="header-button"
                  disabled={!dirty || isSaving || Boolean(outcomeUnknown)}
                  onClick={() => { setReviewKey(reviewKey === lever.key ? null : lever.key); }}
                >
                  Review change
                </button>
              </div>
              {reviewKey === lever.key && dirty && (
                <div className="card" role="group" aria-label={`Review ${lever.key} policy change`}>
                  <p>Review this single-key change before submitting it.</p>
                  <p>Effective now: <code>{lever.value}</code></p>
                  <p>Draft to submit: <code>{draft}</code></p>
                  <p>{LEVER_HELP[lever.key] ?? 'The capacity policy owner validates this value.'} The server response must confirm the effective value.</p>
                  <div className="flex-row-center gap-sm">
                    <button type="button" className="btn btn-action" disabled={isSaving || Boolean(outcomeUnknown)} onClick={() => { onSave(lever.key, draft); setReviewKey(null); }}>Submit this key</button>
                    <button type="button" className="header-button" onClick={() => { setReviewKey(null); }}>Keep editing</button>
                  </div>
                </div>
              )}
              {LEVER_HELP[lever.key] && (
                <span data-sm-style="sm-style-da63c9020c">
                  {LEVER_HELP[lever.key]}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
