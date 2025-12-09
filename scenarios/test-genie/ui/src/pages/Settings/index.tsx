import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "../../components/ui/button";
import {
  fetchPhaseSettings,
  updatePhaseSettings,
  type PhaseSettingsResponse,
  type PhaseToggle
} from "../../lib/api";
import { PHASE_LABELS } from "../../lib/constants";
import { selectors } from "../../consts/selectors";

type PhaseToggleState = Record<string, PhaseToggle>;

const emptyToggle: PhaseToggle = { disabled: false, reason: "", owner: "" };

function normalizeState(payload?: PhaseSettingsResponse): PhaseToggleState {
  const toggles: PhaseToggleState = {};
  if (!payload) return toggles;
  const map = payload.toggles ?? {};
  for (const descriptor of payload.items ?? []) {
    const name = descriptor.name.toLowerCase();
    toggles[name] = {
      ...emptyToggle,
      ...map[name],
      disabled: map[name]?.disabled ?? false
    };
  }
  return toggles;
}

export function SettingsPage() {
  const queryClient = useQueryClient();
  const { data, isLoading, isError } = useQuery({
    queryKey: ["phase-settings"],
    queryFn: fetchPhaseSettings
  });
  const [phaseState, setPhaseState] = useState<PhaseToggleState>({});

  useEffect(() => {
    setPhaseState(normalizeState(data));
  }, [data]);

  const mutation = useMutation({
    mutationFn: (phases: PhaseToggleState) => updatePhaseSettings(phases),
    onSuccess: (payload) => {
      const normalized = normalizeState(payload);
      setPhaseState(normalized);
      queryClient.setQueryData(["phase-settings"], payload);
    }
  });

  const descriptors = data?.items ?? [];
  const disabledCount = useMemo(
    () => Object.values(phaseState).filter((phase) => phase.disabled).length,
    [phaseState]
  );

  const hasValidationError = useMemo(
    () =>
      descriptors.some((descriptor) => {
        const toggle = phaseState[descriptor.name.toLowerCase()];
        return toggle?.disabled && !toggle?.reason?.trim();
      }),
    [descriptors, phaseState]
  );

  const handleToggle = (name: string, disabled: boolean) => {
    const key = name.toLowerCase();
    setPhaseState((prev) => ({
      ...prev,
      [key]: { ...emptyToggle, ...prev[key], disabled }
    }));
  };

  const handleFieldChange = (name: string, field: "reason" | "owner", value: string) => {
    const key = name.toLowerCase();
    setPhaseState((prev) => ({
      ...prev,
      [key]: { ...emptyToggle, ...prev[key], [field]: value }
    }));
  };

  const handleSave = () => {
    mutation.mutate(phaseState);
  };

  const handleReset = () => setPhaseState(normalizeState(data));

  return (
    <div className="flex flex-col gap-4" data-testid={selectors.settings.panel}>
      <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 shadow-lg">
        <p className="text-xs uppercase tracking-[0.35em] text-slate-400">Global safety rails</p>
        <div className="mt-2 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-white">Phase toggles</h1>
            <p className="text-sm text-slate-300">
              Disabled phases are skipped for presets and &ldquo;run all&rdquo; to keep agents focused.
              Explicit requests still run them, but we inject prominent warnings.
            </p>
          </div>
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-amber-100" data-testid={selectors.settings.warning}>
            <p className="text-sm font-semibold">Heads-up</p>
            <p className="text-xs text-amber-100/90">
              Keep reasons current and re-enable phases quickly. Disabled phases shrink coverage.
            </p>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-3 text-sm text-slate-300">
        <span className="rounded-full border border-white/15 bg-white/5 px-3 py-1">
          {descriptors.length} phases tracked
        </span>
        <span className="rounded-full border border-amber-400/30 bg-amber-400/10 px-3 py-1">
          {disabledCount} disabled by default
        </span>
      </div>

      <div className="grid gap-3">
        {isLoading && <div className="text-slate-300">Loading phase settings…</div>}
        {isError && (
          <div className="rounded-xl border border-red-400/40 bg-red-400/10 px-4 py-3 text-red-100">
            Failed to load settings. Retry from the lifecycle console and ensure the API is running.
          </div>
        )}
        {!isLoading &&
          !isError &&
          descriptors.map((descriptor) => {
            const key = descriptor.name.toLowerCase();
            const toggle = phaseState[key] ?? emptyToggle;
            const label = PHASE_LABELS[key] ?? descriptor.name;
            return (
              <div
                key={descriptor.name}
                className="rounded-xl border border-white/10 bg-white/[0.02] p-4 shadow-sm transition hover:border-white/25"
              >
                <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h2 className="text-lg font-semibold text-white">{label}</h2>
                      <span className="rounded-full border border-white/15 bg-white/5 px-2 py-0.5 text-[11px] uppercase tracking-widest text-slate-300">
                        {descriptor.name}
                      </span>
                    </div>
                    {descriptor.description && (
                      <p className="text-sm text-slate-300">{descriptor.description}</p>
                    )}
                  </div>
                  <label className="flex items-center gap-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      className="h-4 w-4 accent-amber-400"
                      checked={toggle.disabled}
                      onChange={(e) => handleToggle(descriptor.name, e.target.checked)}
                    />
                    Disable by default
                  </label>
                </div>

                {toggle.disabled && (
                  <div className="mt-3 grid gap-3 md:grid-cols-2">
                    <label className="flex flex-col gap-1 text-sm text-slate-200">
                      Reason (required for disabled phases)
                      <textarea
                        className="min-h-[72px] rounded-lg border border-amber-500/30 bg-white/5 px-3 py-2 text-sm text-white outline-none focus:border-amber-400"
                        placeholder="Why is this phase disabled? What should agents know before re-enabling?"
                        value={toggle.reason ?? ""}
                        onChange={(e) => handleFieldChange(descriptor.name, "reason", e.target.value)}
                      />
                    </label>
                    <label className="flex flex-col gap-1 text-sm text-slate-200">
                      Owner / approver
                      <input
                        className="rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-white outline-none focus:border-white/40"
                        placeholder="team or person accountable"
                        value={toggle.owner ?? ""}
                        onChange={(e) => handleFieldChange(descriptor.name, "owner", e.target.value)}
                      />
                      <span className="text-xs text-slate-400">Explicit runs will cite this owner.</span>
                    </label>
                    {toggle.addedAt && (
                      <p className="md:col-span-2 text-xs text-slate-400">
                        Disabled since {new Date(toggle.addedAt).toISOString().slice(0, 10)}
                      </p>
                    )}
                    <p className="md:col-span-2 text-sm text-amber-200">
                      Explicit phase requests will still execute and log a warning with this context.
                    </p>
                  </div>
                )}
              </div>
            );
          })}
      </div>

      <div className="flex flex-col gap-2 border-t border-white/10 pt-3 md:flex-row md:items-center md:justify-end">
        <div className="text-sm text-slate-300">
          Saving updates applies globally across all scenarios.
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={handleReset} disabled={mutation.isLoading || isLoading} data-testid={selectors.settings.resetButton}>
            Reset
          </Button>
          <Button
            onClick={handleSave}
            disabled={mutation.isLoading || isLoading || isError || hasValidationError}
            data-testid={selectors.settings.saveButton}
          >
            {mutation.isLoading ? "Saving…" : hasValidationError ? "Add reason to save" : "Save settings"}
          </Button>
        </div>
      </div>
    </div>
  );
}
