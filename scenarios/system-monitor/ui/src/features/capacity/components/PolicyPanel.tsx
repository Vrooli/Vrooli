import { useEffect, useState } from 'react';
import type { PolicyLever } from '../types';

interface PolicyPanelProps {
  levers: PolicyLever[];
  isSaving: boolean;
  error: string | null;
  onSave: (key: string, value: string) => void;
}

const ENFORCE_OPTIONS = ['off', 'advisory', 'on'];

const LEVER_HELP: Record<string, string> = {
  tracking_threshold: 'Min observed bytes before a consumer is reconciled.',
  idle_grace: 'How long a claim must be idle before it is reclaim-eligible.',
  default_heartbeat_ttl: 'Default claim liveness TTL when none is supplied.',
  reconcile_warn_threshold: 'Over-claim drift bytes before a warning is raised.',
  enforce: 'Admission mode: off | advisory | on. Advisory never blocks start.',
  preempt_enabled: 'Whether the last-rung preempt (stop) is permitted.',
  auto_stop_allowlist: 'Owners eligible for auto-stop (comma-separated).',
};

/**
 * Editable policy levers. Each row holds local draft state; Save persists one
 * lever and the parent re-fetches the full validated policy. The enforce lever
 * is a constrained select since it is an enum.
 */
export const PolicyPanel = ({ levers, isSaving, error, onSave }: PolicyPanelProps) => {
  const [drafts, setDrafts] = useState<Record<string, string>>({});

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
      {error && (
        <div role="alert" data-sm-style="sm-style-61375b27ab">
          {error}
        </div>
      )}
      <div data-sm-style="sm-style-9f304072b4">
        {levers.map((lever) => {
          const draft = drafts[lever.key] ?? lever.value;
          const dirty = draft !== lever.value;
          const inputId = `policy-${lever.key}`;
          return (
            <div key={lever.key} data-sm-style="sm-style-a4cd9cd628">
              <label htmlFor={inputId} data-sm-style="sm-style-37fc90e1c1">
                {lever.key}
              </label>
              <div data-sm-style="sm-style-070fbdbe2e">
                {lever.key === 'enforce' ? (
                  <select
                    id={inputId}
                    value={draft}
                    onChange={(e) => { updateDraft(lever.key, e.target.value); }}
                    data-sm-style="sm-style-76aec44f93"
                  >
                    {ENFORCE_OPTIONS.map((opt) => (
                      <option key={opt} value={opt}>{opt}</option>
                    ))}
                  </select>
                ) : (
                  <input
                    id={inputId}
                    type="text"
                    value={draft}
                    onChange={(e) => { updateDraft(lever.key, e.target.value); }}
                    data-sm-style="sm-style-76aec44f93"
                  />
                )}
                <button
                  type="button"
                  className="header-button"
                  disabled={!dirty || isSaving}
                  onClick={() => { onSave(lever.key, draft); }}
                >
                  Save
                </button>
              </div>
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
