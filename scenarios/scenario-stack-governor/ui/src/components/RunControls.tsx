import { Button } from "./ui/button";

export function RunControls({
  disabled,
  isPending,
  onRun
}: {
  disabled: boolean;
  isPending: boolean;
  onRun: () => void;
}) {
  return (
    <div className="flex items-center gap-3">
      {disabled && !isPending && (
        <p className="text-xs text-slate-400">Select at least one rule to run.</p>
      )}
      <Button disabled={disabled || isPending} onClick={onRun}>
        {isPending ? "Running..." : "Run now"}
      </Button>
    </div>
  );
}
