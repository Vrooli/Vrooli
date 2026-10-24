import { ExternalLink } from "lucide-react";
import { classifyNextAction, nextActionLabel } from "../../../lib/consoleActions";
import type { NextAction } from "../../../types/console";

export interface NextActionHandlers {
  onResume?: () => void;
  onRecovery?: () => void;
  onReplan?: () => void;
  onInspectOperation?: () => void;
}

/**
 * NextActionControl renders an API-provided next_action as the control the
 * browser can honour: a link for handoffs and sign-in, a button when this
 * page owns the target view, plain text when the API asks the operator to
 * wait or names something the browser cannot run.
 */
export function NextActionControl({
  next,
  handlers = {},
  testId = "console-next-action",
}: {
  next: NextAction | null | undefined;
  handlers?: NextActionHandlers;
  testId?: string;
}) {
  if (!next) return null;
  const kind = classifyNextAction(next);
  const label = nextActionLabel(next);
  const linkClass =
    "inline-flex items-center gap-1.5 rounded-lg border border-blue-400/40 bg-blue-500/10 px-3 py-2 text-sm font-medium text-blue-100 hover:bg-blue-500/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400";

  if ((kind === "resume_handoff" || kind === "sign_in") && next.reference) {
    return (
      <a href={next.reference} data-testid={testId} data-kind={kind} className={linkClass} rel="noopener noreferrer">
        <ExternalLink className="h-4 w-4" aria-hidden="true" />
        {label}
      </a>
    );
  }

  const handler =
    kind === "resume"
      ? handlers.onResume
      : kind === "recovery"
        ? handlers.onRecovery
        : kind === "replan"
          ? handlers.onReplan
          : kind === "operation"
            ? handlers.onInspectOperation
            : undefined;

  if (handler) {
    return (
      <button type="button" onClick={handler} data-testid={testId} data-kind={kind} className={linkClass}>
        {label}
      </button>
    );
  }

  return (
    <p data-testid={testId} data-kind={kind} className="text-sm text-slate-200">
      {label}
      {next.owner && <span className="text-slate-400"> (owner: {next.owner})</span>}
    </p>
  );
}
