import type { FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight } from "lucide-react";
import { Button } from "../ui/button";
import { InfoTip } from "../cards/InfoTip";
import { selectors } from "../../consts/selectors";
import { useUIStore } from "../../stores/uiStore";
import { queueSuiteRequest } from "../../lib/api";
import { clampCoverage } from "../../lib/formatters";
import { REQUESTED_TYPE_OPTIONS, PRIORITY_OPTIONS } from "../../lib/constants";
import { cn } from "../../lib/utils";

interface QueueFormProps {
  scenarioOptions: string[];
  datalistId: string;
  onSuccess?: () => void;
}

export function QueueForm({ scenarioOptions, datalistId, onSuccess }: QueueFormProps) {
  const queryClient = useQueryClient();
  const {
    queueForm,
    setQueueForm,
    resetQueueForm,
    toggleRequestedType,
    queueFeedback,
    setQueueFeedback
  } = useUIStore();

  const queueMutation = useMutation({
    mutationFn: queueSuiteRequest,
    onMutate: () => setQueueFeedback(null),
    onSuccess: (result) => {
      setQueueFeedback(`Test request queued for ${result.scenarioName}`);
      queryClient.invalidateQueries({ queryKey: ["suite-requests"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
      setQueueForm({
        scenarioName: "",
        notes: "",
        requestedTypes: queueForm.requestedTypes.length ? queueForm.requestedTypes : ["unit", "integration"]
      });
      onSuccess?.();
    },
    onError: (err: Error) => {
      setQueueFeedback(err.message);
    }
  });

  const handleSubmit = (evt: FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!queueForm.scenarioName.trim()) {
      setQueueFeedback("Scenario name is required");
      return;
    }
    const requestedTypes =
      queueForm.requestedTypes.length > 0 ? queueForm.requestedTypes : ["unit", "integration"];
    queueMutation.mutate({
      scenarioName: queueForm.scenarioName.trim(),
      requestedTypes,
      coverageTarget: clampCoverage(queueForm.coverageTarget),
      priority: queueForm.priority,
      notes: queueForm.notes?.trim()
    });
  };

  return (
    <section
      className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
      data-testid={selectors.forms.queueForm}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test Generation</p>
          <div className="mt-2 flex items-center gap-2">
            <h2 className="text-2xl font-semibold">Request tests</h2>
            <InfoTip
              title="Test requests"
              description="Describe what tests you need and Test Genie will generate or queue them for execution."
            />
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Specify the scenario, test types, and target coverage.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={resetQueueForm}
          disabled={queueMutation.isPending}
        >
          Reset
        </Button>
      </div>
      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        <label className="block text-sm">
          <span className="text-slate-300">Scenario name</span>
          <input
            className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            value={queueForm.scenarioName}
            onChange={(evt) => setQueueForm({ scenarioName: evt.target.value })}
            placeholder="e.g. ecosystem-manager"
            list={datalistId}
          />
        </label>

        <label className="block text-sm">
          <span className="text-slate-300">Target pass rate (%)</span>
          <input
            type="number"
            min={1}
            max={100}
            className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            value={queueForm.coverageTarget}
            onChange={(evt) => setQueueForm({ coverageTarget: Number(evt.target.value) })}
          />
        </label>

        <fieldset className="text-sm">
          <legend className="text-slate-300">Test types</legend>
          <div className="mt-3 flex flex-wrap gap-2">
            {REQUESTED_TYPE_OPTIONS.map((type) => (
              <button
                type="button"
                key={type}
                onClick={() => toggleRequestedType(type)}
                className={cn(
                  "rounded-full border px-4 py-2 text-sm transition",
                  queueForm.requestedTypes.includes(type)
                    ? "border-cyan-400 bg-cyan-400/20 text-white"
                    : "border-white/20 text-slate-300 hover:border-white/50"
                )}
              >
                {type}
              </button>
            ))}
          </div>
        </fieldset>

        <label className="block text-sm">
          <span className="text-slate-300">Priority</span>
          <div className="mt-3 flex flex-wrap gap-2">
            {PRIORITY_OPTIONS.map((priority) => (
              <button
                type="button"
                key={priority}
                onClick={() => setQueueForm({ priority })}
                className={cn(
                  "rounded-full border px-4 py-2 text-sm capitalize transition",
                  queueForm.priority === priority
                    ? "border-purple-400 bg-purple-400/20 text-white"
                    : "border-white/20 text-slate-300 hover:border-white/50"
                )}
              >
                {priority}
              </button>
            ))}
          </div>
        </label>

        <label className="block text-sm">
          <span className="text-slate-300">Notes (optional)</span>
          <textarea
            className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            rows={3}
            value={queueForm.notes}
            onChange={(evt) => setQueueForm({ notes: evt.target.value })}
            placeholder="Additional context for test generation"
          />
        </label>

        <Button
          className="w-full"
          type="submit"
          disabled={queueMutation.isPending}
          data-testid={selectors.forms.submitQueue}
        >
          {queueMutation.isPending ? "Submitting..." : "Request tests"}
          <ArrowUpRight className="ml-2 h-4 w-4" />
        </Button>
        {queueFeedback && (
          <p className={cn("text-sm", queueMutation.isError ? "text-red-400" : "text-emerald-300")}>
            {queueFeedback}
          </p>
        )}
      </form>
    </section>
  );
}
