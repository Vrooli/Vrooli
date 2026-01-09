import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { PHASES_FOR_GENERATION } from "../../lib/constants";
import type { TaskType } from "./ScenarioTargetDialog";

interface PhaseSelectorProps {
  selectedPhases: string[];
  onTogglePhase: (phase: string) => void;
  lockToUnit?: boolean;
  task?: TaskType | null;
}

function getPhaseLabel(phaseKey: string, task: TaskType | null): string {
  const baseLabels: Record<string, Record<TaskType | "default", string>> = {
    unit: {
      bootstrap: "Bootstrap unit tests",
      coverage: "Add unit test coverage",
      "fix-failing": "Fix failing unit tests",
      default: "Unit Tests"
    },
    playbooks: {
      bootstrap: "Bootstrap E2E playbooks",
      coverage: "Add E2E playbook coverage",
      "fix-failing": "Fix failing E2E playbooks",
      default: "E2E Playbooks"
    }
  };
  const labels = baseLabels[phaseKey];
  if (!labels) return phaseKey;
  return task ? (labels[task] ?? labels.default) : labels.default;
}

function getPhaseDescription(phaseKey: string, task: TaskType | null): string {
  const baseDescriptions: Record<string, Record<TaskType | "default", string>> = {
    unit: {
      bootstrap: "Create initial unit tests for functions and modules",
      coverage: "Add unit tests for specific features or edge cases",
      "fix-failing": "Fix and improve failing unit tests",
      default: "Unit tests for individual functions and modules"
    },
    playbooks: {
      bootstrap: "Create initial E2E browser automation workflows",
      coverage: "Add E2E playbooks for specific user flows",
      "fix-failing": "Fix and improve failing E2E playbooks",
      default: "End-to-end browser automation workflows"
    }
  };
  const descriptions = baseDescriptions[phaseKey];
  if (!descriptions) return "";
  return task ? (descriptions[task] ?? descriptions.default) : descriptions.default;
}

export function PhaseSelector({ selectedPhases, onTogglePhase, lockToUnit, task }: PhaseSelectorProps) {
  return (
    <div data-testid={selectors.generate.phaseSelector}>
      <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test phases</p>
      <h3 className="mt-2 text-lg font-semibold">Select phases</h3>
      <p className="mt-2 text-sm text-slate-300">
        Choose which test types to include.
        {lockToUnit && " Targeting specific paths locks generation to Unit tests for safety and parallelism."}
      </p>

      <div className="mt-4 grid gap-3 md:grid-cols-2">
        {PHASES_FOR_GENERATION.map((phase) => (
          <button
            key={phase.key}
            type="button"
            onClick={() => {
              if (lockToUnit && phase.key !== "unit") return;
              onTogglePhase(phase.key);
            }}
            className={cn(
              "rounded-xl border p-4 text-left transition",
              selectedPhases.includes(phase.key)
                ? "border-cyan-400 bg-cyan-400/10"
                : "border-white/10 bg-black/30 hover:border-white/30",
              lockToUnit && phase.key !== "unit" && "opacity-50 cursor-not-allowed"
            )}
            disabled={lockToUnit && phase.key !== "unit"}
          >
            <div className="flex items-center justify-between">
              <span className="font-semibold">{getPhaseLabel(phase.key, task ?? null)}</span>
              <span
                className={cn(
                  "h-5 w-5 rounded border flex items-center justify-center",
                  selectedPhases.includes(phase.key)
                    ? "border-cyan-400 bg-cyan-400 text-black"
                    : "border-white/30"
                )}
              >
                {selectedPhases.includes(phase.key) && (
                  <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
                    <path
                      fillRule="evenodd"
                      d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                )}
              </span>
            </div>
            <p className="mt-2 text-xs text-slate-400">
              {getPhaseDescription(phase.key, task ?? null)}
            </p>
          </button>
        ))}
      </div>
    </div>
  );
}
