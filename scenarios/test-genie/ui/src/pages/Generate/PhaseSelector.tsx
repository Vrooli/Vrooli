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

type PhaseTaskCopy = Partial<Record<TaskType, { label: string; description: string }>>;

const phaseTaskCopy: Record<string, PhaseTaskCopy> = {
  unit: {
    bootstrap: {
      label: "Bootstrap unit tests",
      description: "Create initial unit tests for functions and modules"
    },
    coverage: {
      label: "Add unit test coverage",
      description: "Add unit tests for specific features or edge cases"
    },
    "fix-failing": {
      label: "Fix failing unit tests",
      description: "Fix and improve failing unit tests"
    }
  },
  playbooks: {
    bootstrap: {
      label: "Bootstrap E2E playbooks",
      description: "Create initial E2E browser automation workflows"
    },
    coverage: {
      label: "Add E2E playbook coverage",
      description: "Add E2E playbooks for specific user flows"
    },
    "fix-failing": {
      label: "Fix failing E2E playbooks",
      description: "Fix and improve failing E2E playbooks"
    }
  }
};

function getPhaseCopy(
  phase: (typeof PHASES_FOR_GENERATION)[number],
  task: TaskType | null
): { label: string; description: string } {
  if (!task) {
    return { label: phase.label, description: phase.description };
  }
  const taskSpecific = phaseTaskCopy[phase.key]?.[task];
  if (taskSpecific) {
    return taskSpecific;
  }
  return { label: phase.label, description: phase.description };
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
        {PHASES_FOR_GENERATION.map((phase) => {
          const copy = getPhaseCopy(phase, task ?? null);
          const isSelected = selectedPhases.includes(phase.key);
          const isLocked = Boolean(lockToUnit && phase.key !== "unit");

          return (
            <button
              key={phase.key}
              type="button"
              onClick={() => {
                if (isLocked) return;
                onTogglePhase(phase.key);
              }}
              className={cn(
                "rounded-xl border p-4 text-left transition",
                isSelected
                  ? "border-cyan-400 bg-cyan-400/10"
                  : "border-white/10 bg-black/30 hover:border-white/30",
                isLocked && "opacity-50 cursor-not-allowed"
              )}
              disabled={isLocked}
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">{copy.label}</span>
                <span
                  className={cn(
                    "h-5 w-5 rounded border flex items-center justify-center",
                    isSelected
                      ? "border-cyan-400 bg-cyan-400 text-black"
                      : "border-white/30"
                  )}
                >
                  {isSelected && (
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
              <p className="mt-2 text-xs text-slate-400">{copy.description}</p>
            </button>
          );
        })}
      </div>
    </div>
  );
}
