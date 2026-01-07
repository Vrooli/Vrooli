import { useState } from "react";
import { HelpCircle, ChevronUp, ChevronDown, Check, X, Circle } from "lucide-react";
import { selectors } from "../../consts/selectors";

export function RequirementsHelpSection() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div
      className="rounded-xl border border-white/10 bg-white/[0.02] px-4 py-3"
      data-testid={selectors.requirements.helpSection}
    >
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex w-full items-center gap-2 text-sm text-slate-400 transition hover:text-white"
        data-testid={selectors.requirements.helpToggle}
      >
        <HelpCircle className="h-4 w-4" />
        <span>How requirements pass or fail</span>
        {isOpen ? (
          <ChevronUp className="ml-auto h-4 w-4" />
        ) : (
          <ChevronDown className="ml-auto h-4 w-4" />
        )}
      </button>

      {isOpen && (
        <div className="mt-4 space-y-3 text-sm text-slate-300">
          <div className="flex items-start gap-3">
            <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
            <div>
              <p className="font-medium text-emerald-400">Passed</p>
              <p className="text-slate-400">
                Requirement status is &quot;complete&quot; AND all validations
                are &quot;implemented&quot; or &quot;passing&quot;
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <X className="mt-0.5 h-4 w-4 shrink-0 text-red-400" />
            <div>
              <p className="font-medium text-red-400">Failed</p>
              <p className="text-slate-400">
                Any validation has status &quot;failing&quot; - even if the
                requirement is marked complete
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <Circle className="mt-0.5 h-4 w-4 shrink-0 text-amber-400" />
            <div>
              <p className="font-medium text-amber-400">Not Run</p>
              <p className="text-slate-400">
                Requirement is &quot;pending&quot; or &quot;in_progress&quot;,
                or has no validations defined
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
