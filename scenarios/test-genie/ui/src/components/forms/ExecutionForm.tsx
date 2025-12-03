import type { FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight } from "lucide-react";
import { Button } from "../ui/button";
import { InfoTip } from "../cards/InfoTip";
import { selectors } from "../../consts/selectors";
import { useUIStore } from "../../stores/uiStore";
import { triggerSuiteExecution } from "../../lib/api";
import { EXECUTION_PRESETS, PRESET_DETAILS, PHASE_LABELS } from "../../lib/constants";
import { cn } from "../../lib/utils";

interface ExecutionFormProps {
  scenarioOptions: string[];
  datalistId: string;
  onSuccess?: () => void;
}

export function ExecutionForm({ scenarioOptions, datalistId, onSuccess }: ExecutionFormProps) {
  const queryClient = useQueryClient();
  const {
    executionForm,
    setExecutionForm,
    resetExecutionForm,
    executionFeedback,
    setExecutionFeedback
  } = useUIStore();

  const executionMutation = useMutation({
    mutationFn: triggerSuiteExecution,
    onMutate: () => setExecutionFeedback(null),
    onSuccess: (result) => {
      const resultLabel = result.success ? "completed" : "failed";
      setExecutionFeedback(
        `Tests ${resultLabel} in ${(result.phaseSummary.durationSeconds / 60).toFixed(1)} min`
      );
      queryClient.invalidateQueries({ queryKey: ["executions"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
      onSuccess?.();
    },
    onError: (err: Error) => {
      setExecutionFeedback(err.message);
    }
  });

  const handleSubmit = (evt: FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!executionForm.scenarioName.trim()) {
      setExecutionFeedback("Scenario name is required");
      return;
    }
    executionMutation.mutate({
      scenarioName: executionForm.scenarioName.trim(),
      preset: executionForm.preset,
      failFast: executionForm.failFast,
      suiteRequestId: executionForm.suiteRequestId.trim() || undefined
    });
  };

  const presetEntries = Object.entries(PRESET_DETAILS) as Array<
    [string, (typeof PRESET_DETAILS)[keyof typeof PRESET_DETAILS]]
  >;

  return (
    <section
      className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
      data-testid={selectors.forms.executionForm}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test Runner</p>
          <div className="mt-2 flex items-center gap-2">
            <h2 className="text-2xl font-semibold">Run tests</h2>
            <InfoTip
              title="Run tests"
              description="Execute tests for a scenario using a preset configuration."
            />
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Select a preset to run different test phases.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={resetExecutionForm}
          disabled={executionMutation.isPending}
        >
          Reset
        </Button>
      </div>
      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        <label className="block text-sm">
          <span className="text-slate-300">Scenario name</span>
          <input
            className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            value={executionForm.scenarioName}
            onChange={(evt) => setExecutionForm({ scenarioName: evt.target.value })}
            placeholder="scenario-under-test"
            list={datalistId}
          />
        </label>

        <label className="block text-sm">
          <span className="text-slate-300">Preset</span>
          <div className="mt-3 flex flex-wrap gap-2">
            {EXECUTION_PRESETS.map((preset) => (
              <button
                type="button"
                key={preset}
                onClick={() => setExecutionForm({ preset })}
                className={cn(
                  "rounded-full border px-4 py-2 text-sm capitalize transition",
                  executionForm.preset === preset
                    ? "border-emerald-400 bg-emerald-400/20 text-white"
                    : "border-white/20 text-slate-300 hover:border-white/50"
                )}
              >
                {preset}
              </button>
            ))}
          </div>
        </label>

        <label className="inline-flex items-center gap-3 text-sm text-slate-300">
          <input
            type="checkbox"
            checked={executionForm.failFast}
            onChange={(evt) => setExecutionForm({ failFast: evt.target.checked })}
            className="h-5 w-5 rounded border border-white/20 bg-black/30"
          />
          Stop on first failed phase
        </label>

        <label className="block text-sm">
          <span className="text-slate-300">Link to request (optional)</span>
          <input
            className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            value={executionForm.suiteRequestId}
            onChange={(evt) => setExecutionForm({ suiteRequestId: evt.target.value })}
            placeholder="Link to queue row for status updates"
          />
        </label>

        <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Preset phases</p>
          <p className="mt-1 text-sm text-slate-300">
            Each preset runs different test phases.
          </p>
          <div className="mt-4 grid gap-3 lg:grid-cols-3">
            {presetEntries.map(([presetKey, detail]) => (
              <div
                key={presetKey}
                className={cn(
                  "rounded-xl border bg-white/5 p-3 text-sm",
                  executionForm.preset === presetKey
                    ? "border-emerald-400/60 bg-emerald-400/10"
                    : "border-white/10 bg-black/30"
                )}
              >
                <p className="font-semibold capitalize">{detail.label}</p>
                <p className="mt-1 text-xs text-slate-400">{detail.description}</p>
                <ul className="mt-3 space-y-1 text-xs text-slate-300">
                  {detail.phases.map((phase) => (
                    <li key={`${presetKey}-${phase}`} className="flex items-center gap-2">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                      <span>{PHASE_LABELS[phase] ?? phase}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <Button
          className="w-full"
          type="submit"
          disabled={executionMutation.isPending}
          data-testid={selectors.forms.submitExecution}
        >
          {executionMutation.isPending ? "Running..." : "Run tests"}
          <ArrowUpRight className="ml-2 h-4 w-4" />
        </Button>
        {executionFeedback && (
          <p
            className={cn(
              "text-sm",
              executionMutation.isError
                ? "text-red-400"
                : executionMutation.isSuccess
                  ? "text-emerald-300"
                  : "text-slate-300"
            )}
          >
            {executionFeedback}
          </p>
        )}
      </form>
    </section>
  );
}
