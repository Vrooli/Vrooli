import { SectionPanel } from "./Surfaces";
import { shellCommand } from "../lib/shellCommand";
import type { GuidedStep } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

export function GuidedStepPanel({
  step,
  headingId,
  testId,
  commandPrefix,
}: {
  step?: GuidedStep;
  headingId: string;
  testId: string;
  commandPrefix?: readonly string[];
}) {
  if (!step || (!step.title && !step.summary)) return null;
  return (
    <SectionPanel title={step.title || step.stepKind} headingId={headingId}>
      <div data-testid={testId} className="flex flex-col gap-3">
        {step.summary ? <p className="text-sm text-app-muted-foreground">{step.summary}</p> : null}
        {step.instructions.length > 0 ? (
          <ul className="flex flex-col gap-1 text-sm text-app-foreground">
            {step.instructions.map((item, i) => (
              <li key={`${item}-${i}`}>{item}</li>
            ))}
          </ul>
        ) : null}
        {step.requiredInputs.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {step.requiredInputs.map((item) => (
              <span
                key={item}
                className="rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs"
              >
                {item}
              </span>
            ))}
          </div>
        ) : null}
        {step.nextActions.length > 0 ? (
          <div className="flex flex-col gap-2">
            {step.nextActions.map((action) => (
              <code
                key={action.id || `${action.label}-${action.argv.join(" ")}`}
                className="break-all rounded-control bg-app-surface-muted px-3 py-2 text-xs text-app-foreground"
              >
                {shellCommand(action.argv, commandPrefix)}
              </code>
            ))}
          </div>
        ) : null}
      </div>
    </SectionPanel>
  );
}
