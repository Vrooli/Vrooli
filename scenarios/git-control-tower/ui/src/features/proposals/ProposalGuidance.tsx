import type { ReactNode } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { HelpCircle } from "lucide-react";

export interface GuidanceAction {
  label: string;
  onClick: () => void;
}

/** A blocked native button stays protected. Its wrapper owns explanation,
 * keyboard activation and touch input instead of leaving a dead control. */
export function GuidedCommitButton({
  blocked,
  reason,
  onExplain,
  onClick,
  children,
  testId,
  className,
}: {
  blocked: boolean;
  reason?: string;
  onExplain: () => void;
  onClick: () => void;
  children: ReactNode;
  testId: string;
  className: string;
}) {
  const button = (
    <button
      type="button"
      className={`${className} ${blocked ? "pointer-events-none" : ""}`}
      disabled={blocked}
      aria-hidden={blocked || undefined}
      onClick={onClick}
      data-testid={testId}
    >
      {children}
      {blocked && <HelpCircle className="h-3.5 w-3.5" aria-hidden="true" />}
    </button>
  );
  if (!blocked) return button;
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={`Explain why Create commit is unavailable: ${reason || "explanation available"}`}
      title="Click for an explanation and next steps"
      aria-haspopup="dialog"
      className="cursor-help rounded-full outline-offset-4 focus-visible:outline focus-visible:outline-cyan-400"
      data-testid={`${testId}-explain`}
      onClick={onExplain}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onExplain();
        }
      }}
    >
      {button}
    </div>
  );
}

export function ProposalGuidance({
  title,
  children,
  actions,
  onClose,
  pending = false,
  error,
}: {
  title: string;
  children: ReactNode;
  actions: GuidanceAction[];
  onClose: () => void;
  pending?: boolean;
  error?: string;
}) {
  return (
    <ResponsiveDialog
      open
      onClose={onClose}
      title={
        <span className="flex items-center gap-2 text-slate-100">
          <HelpCircle className="h-5 w-5 text-cyan-300" />
          {title}
        </span>
      }
      closeLabel="Close explanation"
      ariaLabel={title}
      size="lg"
      contentPadding="comfortable"
      avoidKeyboard
      testId="proposal-guidance"
      panelClassName="!border-slate-700 !bg-slate-900 !text-slate-200"
      footer={
        <div className="flex w-full flex-col gap-2 sm:flex-row-reverse sm:flex-wrap">
          {actions.map((action, index) => (
            <button
              type="button"
              key={action.label}
              disabled={pending}
              className={`min-h-11 rounded-lg px-4 py-2 text-sm disabled:opacity-50 ${index === 0 ? "bg-blue-600 text-white hover:bg-blue-500" : "border border-slate-700 text-slate-200 hover:bg-slate-800"}`}
              onClick={action.onClick}
            >
              {action.label}
            </button>
          ))}
          <button
            type="button"
            className="min-h-11 rounded-lg px-4 py-2 text-sm text-slate-400 hover:bg-slate-800"
            onClick={onClose}
          >
            Back to review
          </button>
        </div>
      }
    >
      <div className="space-y-4 text-sm leading-relaxed text-slate-300">
        {error && (
          <p
            role="alert"
            className="rounded-lg border border-red-800 bg-red-950/30 p-3 text-red-200"
          >
            {error}
          </p>
        )}
        {children}
      </div>
    </ResponsiveDialog>
  );
}
