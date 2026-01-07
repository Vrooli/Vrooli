import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { ImproveActionType } from "../../lib/api";

interface ActionTypeSelectorProps {
  value: ImproveActionType;
  onChange: (value: ImproveActionType) => void;
  disabled?: boolean;
}

const ACTION_OPTIONS: Array<{
  value: ImproveActionType;
  label: string;
  description: string;
}> = [
  {
    value: "write_tests",
    label: "Write/Fix Tests",
    description: "Create or fix tests for requirements"
  },
  {
    value: "update_requirements",
    label: "Update Requirements",
    description: "Update requirement files to match implementation"
  },
  {
    value: "both",
    label: "Both",
    description: "Write tests and update requirements"
  }
];

export function ActionTypeSelector({
  value,
  onChange,
  disabled
}: ActionTypeSelectorProps) {
  return (
    <div
      className="flex flex-col gap-2"
      data-testid={selectors.requirements.actionTypeSelector}
    >
      <label className="text-xs text-slate-400">Action Type</label>
      <div className="flex flex-wrap gap-2">
        {ACTION_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            type="button"
            onClick={() => onChange(opt.value)}
            disabled={disabled}
            className={cn(
              "rounded-lg px-3 py-2 text-sm transition",
              value === opt.value
                ? "border border-violet-500/50 bg-violet-500/20 text-violet-300"
                : "border border-white/10 bg-black/30 text-slate-400 hover:border-white/20 hover:text-white",
              disabled && "cursor-not-allowed opacity-50"
            )}
            title={opt.description}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </div>
  );
}
