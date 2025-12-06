import { FormEvent, useEffect, useState } from "react";
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
  scenarioName?: string;
  onSuccess?: () => void;
}

export function ExecutionForm({ scenarioOptions, datalistId, scenarioName, onSuccess }: ExecutionFormProps) {
  const queryClient = useQueryClient();
  const {
    executionForm,
    setExecutionForm,
    resetExecutionForm,
    executionFeedback,
    setExecutionFeedback
  } = useUIStore();
  const [feedbackStatus, setFeedbackStatus] = useState<"success" | "error" | null>(null);

  // When a scenario is provided by the parent (e.g., scenario detail page), sync it and clear stale suite request links.
  useEffect(() => {
    if (scenarioName) {
      setExecutionForm({ scenarioName, suiteRequestId: "" });
    }
  }, [scenarioName, setExecutionForm]);

  const executionMutation = useMutation({
    mutationFn: triggerSuiteExecution,
    onMutate: () => {
      setExecutionFeedback(null);
      setFeedbackStatus(null);
    },
    onSuccess: (result) => {
      const mins = ((result.phaseSummary?.durationSeconds ?? 0) / 60).toFixed(1);
      const failing = (result.phases || []).filter((p) => p.status && p.status !== "passed");
      const firstError = failing.find((p) => p.error)?.error;
      const resultLabel = result.success ? "completed" : "failed";
      const errorNote = !result.success
        ? ` · ${failing.length} phase${failing.length === 1 ? "" : "s"} failed${firstError ? `: ${firstError}` : ""}`
        : "";
      setExecutionFeedback(`Tests ${resultLabel} in ${mins} min${errorNote}`);
      setFeedbackStatus(result.success ? "success" : "error");
      queryClient.invalidateQueries({ queryKey: ["executions"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
      onSuccess?.();
    },
    onError: (err: Error) => {
      setExecutionFeedback(err.message);
      setFeedbackStatus("error");
    }
  });

  const handleSubmit = (evt: FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!executionForm.scenarioName.trim()) {
      setExecutionFeedback("Scenario name is required");
      setFeedbackStatus("error");
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
        {!scenarioName && (
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
        )}
        {scenarioName && (
          <div className="flex items-center gap-2 text-sm text-slate-300">
            <span className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-3 py-1 text-emerald-100">
              Scenario: {scenarioName}
            </span>
          </div>
        )}

        <label className="inline-flex items-center gap-3 text-sm text-slate-300">
          <input
            type="checkbox"
            checked={executionForm.failFast}
            onChange={(evt) => setExecutionForm({ failFast: evt.target.checked })}
            className="h-5 w-5 rounded border border-white/20 bg-black/30"
          />
          Stop on first failed phase
        </label>

        <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Preset phases</p>
          <p className="mt-1 text-sm text-slate-300">
            Each preset runs different test phases. Click a card to select.
          </p>
          <div className="mt-4 grid gap-3 lg:grid-cols-3">
            {presetEntries.map(([presetKey, detail]) => (
              <div
                key={presetKey}
                role="button"
                tabIndex={0}
                onClick={() => setExecutionForm({ preset: presetKey })}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    setExecutionForm({ preset: presetKey });
                  }
                }}
                className={cn(
                  "rounded-xl border bg-white/5 p-3 text-sm cursor-pointer focus:outline-none focus:ring-2 focus:ring-cyan-400",
                  executionForm.preset === presetKey
                    ? "border-emerald-400/60 bg-emerald-400/10"
                    : "border-white/10 bg-black/30 hover:border-white/40"
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
            feedbackStatus === "error"
              ? "text-red-400"
              : feedbackStatus === "success"
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
