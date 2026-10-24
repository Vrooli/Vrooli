import { SectionPanel } from "./Surfaces";
import { shellCommand } from "../lib/shellCommand";
import { NextActionKind, type GuidedStep } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

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
  const { t } = useTranslation();
  if (!step || (!step.title && !step.summary)) return null;
  const instructions = arrayOrEmpty(step.instructions);
  const requiredInputs = arrayOrEmpty(step.requiredInputs);
  const examples = arrayOrEmpty(step.examples);
  const actions = arrayOrEmpty(step.nextActions);
  const commonMistakes = arrayOrEmpty(step.commonMistakes);

  return (
    <SectionPanel title={step.title || step.stepKind} headingId={headingId}>
      <div data-testid={testId} className="flex flex-col gap-3">
        {step.summary ? <p className="text-sm text-app-muted-foreground">{step.summary}</p> : null}
        {instructions.length > 0 ? (
          <ul className="flex flex-col gap-1 text-sm text-app-foreground">
            {instructions.map((item, i) => (
              <li key={`${item}-${i}`}>{item}</li>
            ))}
          </ul>
        ) : null}
        {requiredInputs.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {requiredInputs.map((item) => (
              <span
                key={item}
                className="rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs"
              >
                {item}
              </span>
            ))}
          </div>
        ) : null}
        {examples.length > 0 ? (
          <p className="text-xs text-app-muted-foreground">
            <span className="font-medium text-app-foreground">{t(strings.guidedStep.example)}:</span>{" "}
            {examples[0]}
          </p>
        ) : null}
        {actions.length > 0 ? (
          <ul className="flex flex-col gap-2">
            {actions.map((action) => {
              const blockedBy = arrayOrEmpty(action.blockedBy);
              const argv = arrayOrEmpty(action.argv);
              return (
                <li
                  key={action.id || `${action.label}-${argv.join(" ")}`}
                  className="flex flex-col gap-1 rounded-control bg-app-surface-muted px-3 py-2 text-sm text-app-foreground"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="rounded-control border border-app-border px-1.5 py-0.5 text-[0.7rem] uppercase tracking-normal text-app-muted-foreground">
                      {t(actionKindKey(action.kind))}
                    </span>
                    <span className="font-medium">{action.label || t(strings.guidedStep.action)}</span>
                  </div>
                  {action.reason ? <p className="text-xs text-app-muted-foreground">{action.reason}</p> : null}
                  <code className="break-all font-mono text-xs text-app-foreground">
                    {shellCommand(argv, commandPrefix)}
                  </code>
                  {blockedBy.length > 0 ? (
                    <ul className="flex flex-col gap-1 text-xs text-app-danger">
                      {blockedBy.map((item) => (
                        <li key={item}>
                          {t(strings.guidedStep.blocked)}: {item}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </li>
              );
            })}
          </ul>
        ) : null}
        {commonMistakes.length > 0 ? (
          <div className="text-xs text-app-muted-foreground">
            <span className="font-medium text-app-foreground">{t(strings.guidedStep.avoid)}:</span>{" "}
            {commonMistakes.join("; ")}
          </div>
        ) : null}
      </div>
    </SectionPanel>
  );
}

function arrayOrEmpty<T>(value: T[]): T[] {
  return Array.isArray(value) ? value : [];
}

function actionKindKey(kind: NextActionKind) {
  switch (kind) {
    case NextActionKind.RECOMMENDED:
      return strings.guidedStep.recommended;
    case NextActionKind.ALTERNATIVE:
      return strings.guidedStep.alternative;
    case NextActionKind.OPTIONAL:
      return strings.guidedStep.optional;
    case NextActionKind.RECOVERY:
      return strings.guidedStep.recovery;
    default:
      return strings.guidedStep.action;
  }
}
