import { Info } from "lucide-react";
import { Button } from "./button";

interface TabTipProps {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export const TabTip = ({ title, description, actionLabel, onAction }: TabTipProps) => {
  return (
    <div className="flex items-start gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
      <div className="mt-0.5 rounded-full border border-emerald-400/40 bg-emerald-500/10 p-2">
        <Info className="h-4 w-4 text-emerald-300" />
      </div>
      <div className="flex-1">
        <p className="text-sm font-semibold text-white">{title}</p>
        <p className="text-xs text-white/70">{description}</p>
      </div>
      {actionLabel && onAction ? (
        <Button size="sm" variant="secondary" onClick={onAction}>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
};
