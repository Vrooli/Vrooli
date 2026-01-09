import { Circle, CircleDot, AlertTriangle } from "lucide-react";
import type { DeploymentManifestSecret } from "../../lib/api";
import { isSecretBlocking, getStrategyColorClass } from "./utils";

interface SecretListItemProps {
  secret: DeploymentManifestSecret;
  isSelected: boolean;
  isExcluded: boolean;
  isOverridden: boolean;
  onSelect: () => void;
  onToggleExclude: () => void;
}

export function SecretListItem({
  secret,
  isSelected,
  isExcluded,
  isOverridden,
  onSelect,
  onToggleExclude
}: SecretListItemProps) {
  const isBlocking = isSecretBlocking(secret);
  const strategy = secret.handling_strategy || "none";

  return (
    <div
      className={`flex items-center gap-2 px-3 py-1.5 cursor-pointer transition-colors ${
        isExcluded
          ? "bg-white/5 opacity-50"
          : isSelected
          ? "bg-emerald-500/10 border-l-2 border-emerald-400"
          : "hover:bg-white/5"
      }`}
    >
      <input
        type="checkbox"
        checked={!isExcluded}
        onChange={(e) => {
          e.stopPropagation();
          onToggleExclude();
        }}
        className="h-3.5 w-3.5 rounded border-white/20 bg-slate-800 text-emerald-500 focus:ring-emerald-500/50 cursor-pointer"
        title={isExcluded ? "Include in export" : "Exclude from export"}
      />
      <button
        type="button"
        onClick={onSelect}
        className={`flex-1 flex items-center gap-2 text-left min-w-0 ${isExcluded ? "line-through" : ""}`}
      >
        {isOverridden ? (
          <span title="Has scenario override">
            <CircleDot className="h-3 w-3 text-purple-400 shrink-0" />
          </span>
        ) : isBlocking ? (
          <span title="Blocking - no strategy">
            <AlertTriangle className="h-3 w-3 text-amber-400 shrink-0" />
          </span>
        ) : (
          <span title="Using resource default">
            <Circle className="h-3 w-3 text-white/30 shrink-0" />
          </span>
        )}
        <span className="font-mono text-xs text-white/80 truncate">{secret.secret_key}</span>
        <span
          className={`ml-auto shrink-0 rounded-full border px-1.5 py-0.5 text-[10px] ${getStrategyColorClass(
            strategy
          )}`}
        >
          {isBlocking ? "none" : strategy}
        </span>
      </button>
    </div>
  );
}
