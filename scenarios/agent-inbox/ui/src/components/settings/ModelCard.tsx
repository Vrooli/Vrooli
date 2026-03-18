import { Bot, Check, Image, X, Sparkles } from "lucide-react";
import type { Model } from "../../lib/api";
import { supportsImageGeneration } from "../../lib/modelCapabilities";
import { formatPrice, getModalityBadges } from "./modelSelectorUtils";

interface ModelCardProps {
  model: Model;
  isSelected: boolean;
  onSelect: (modelId: string) => void;
  /** If provided, shows a remove button (for recent models) */
  onRemove?: (e: React.MouseEvent, modelId: string) => void;
  testIdPrefix?: string;
}

export function ModelCard({
  model,
  isSelected,
  onSelect,
  onRemove,
  testIdPrefix = "model-option",
}: ModelCardProps) {
  const modalities = getModalityBadges(model);
  const hasImageSupport = modalities.includes("image");
  const canGenerateImages = supportsImageGeneration(model);

  return (
    <button
      onClick={() => onSelect(model.id)}
      className={`w-full flex items-start gap-3 p-3 rounded-lg text-left transition-colors ${
        isSelected
          ? "bg-indigo-500/20 border border-indigo-500/50"
          : "hover:bg-white/5 border border-transparent"
      }`}
      data-testid={`${testIdPrefix}-${model.id}`}
    >
      <div className="shrink-0 mt-0.5">
        <Bot className={`h-5 w-5 ${isSelected ? "text-indigo-400" : "text-slate-400"}`} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={`font-medium ${isSelected ? "text-white" : "text-slate-200"}`}>
            {model.name}
          </span>
          {isSelected && (
            <Check className="h-4 w-4 text-indigo-400 shrink-0" />
          )}
        </div>

        {/* Metadata row */}
        <div className="text-xs text-slate-500 mt-1 flex items-center gap-2 flex-wrap">
          {hasImageSupport && (
            <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300">
              <Image className="h-3 w-3" />
              <span>Vision</span>
            </span>
          )}
          {canGenerateImages && (
            <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-pink-500/20 text-pink-300">
              <Sparkles className="h-3 w-3" />
              <span>Image Gen</span>
            </span>
          )}
          {model.context_length && (
            <span>{(model.context_length / 1000).toFixed(0)}K ctx</span>
          )}
          {model.pricing && (
            <>
              <span className="text-slate-600">&bull;</span>
              <span className="text-emerald-400">
                {formatPrice(model.pricing.prompt)}/M in
              </span>
              <span className="text-slate-600">/</span>
              <span className="text-amber-400">
                {formatPrice(model.pricing.completion)}/M out
              </span>
            </>
          )}
        </div>

        {!onRemove && model.description && (
          <p className="text-sm text-slate-400 mt-1.5 line-clamp-3">
            {model.description}
          </p>
        )}
      </div>

      {onRemove && (
        <div
          role="button"
          tabIndex={0}
          onClick={(e) => onRemove(e, model.id)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              onRemove(e as unknown as React.MouseEvent, model.id);
            }
          }}
          className="shrink-0 p-1.5 rounded-md hover:bg-white/10 text-slate-500 hover:text-slate-300 transition-colors"
          title="Remove from recent"
          data-testid={`remove-recent-${model.id}`}
        >
          <X className="h-4 w-4" />
        </div>
      )}
    </button>
  );
}
