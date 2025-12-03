import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { PHASES_FOR_GENERATION } from "../../lib/constants";

interface PhaseSelectorProps {
  selectedPhases: string[];
  onTogglePhase: (phase: string) => void;
}

export function PhaseSelector({ selectedPhases, onTogglePhase }: PhaseSelectorProps) {
  return (
    <div data-testid={selectors.generate.phaseSelector}>
      <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test phases</p>
      <h3 className="mt-2 text-lg font-semibold">Select phases to generate</h3>
      <p className="mt-2 text-sm text-slate-300">
        Choose which test phases to include in your generation prompt.
      </p>

      <div className="mt-4 grid gap-3 md:grid-cols-2">
        {PHASES_FOR_GENERATION.map((phase) => (
          <button
            key={phase.key}
            type="button"
            onClick={() => onTogglePhase(phase.key)}
            className={cn(
              "rounded-xl border p-4 text-left transition",
              selectedPhases.includes(phase.key)
                ? "border-cyan-400 bg-cyan-400/10"
                : "border-white/10 bg-black/30 hover:border-white/30"
            )}
          >
            <div className="flex items-center justify-between">
              <span className="font-semibold">{phase.label}</span>
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
              {phase.key === "unit" && "Generate unit tests for individual functions and modules"}
              {phase.key === "integration" && "Generate integration tests for component interactions"}
              {phase.key === "playbooks" && "Generate end-to-end browser automation workflows"}
              {phase.key === "business" && "Generate business requirement validation tests"}
            </p>
          </button>
        ))}
      </div>
    </div>
  );
}
