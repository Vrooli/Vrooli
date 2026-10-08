import { Terminal } from "lucide-react";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";

export interface ScenarioCliHintsProps {
  name: string;
  /** Layout variant: mobile uses vertical stack, desktop uses 2-col grid. */
  variant?: "mobile" | "desktop";
}

const CLI_COMMANDS = [
  { label: "View Logs", cmd: "vrooli scenario logs" },
  { label: "Check Status", cmd: "vrooli scenario status" },
  { label: "Run Tests", cmd: "vrooli scenario test" },
  { label: "Restart Scenario", cmd: "vrooli scenario restart" },
];

export function ScenarioCliHints({ name, variant = "desktop" }: ScenarioCliHintsProps) {
  const gridClass = variant === "desktop" ? "grid gap-3 sm:grid-cols-2" : "space-y-2";

  return (
    <DetailSection
      title="Quick Actions (CLI)"
      icon={Terminal}
      storageKey={variant === "mobile" ? "scenario-cli-hints-mobile" : undefined}
      defaultOpen={variant === "desktop"}
      data-testid={selectors.scenarioDetails.cliHint}
    >
      {variant === "desktop" && (
        <p className="mb-4 text-sm text-slate-400">
          Common operations for this scenario are also available via the command line.
        </p>
      )}
      {variant === "mobile" && (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            Common operations for this scenario are also available via the command line.
          </p>
        </div>
      )}
      <div className={gridClass}>
        {CLI_COMMANDS.map(({ label, cmd }) => (
          <div key={label} className="rounded-lg bg-slate-700/30 p-3">
            <span className="text-xs font-medium text-slate-300">{label}</span>
            <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
              {cmd} {name}
            </code>
          </div>
        ))}
      </div>
    </DetailSection>
  );
}
