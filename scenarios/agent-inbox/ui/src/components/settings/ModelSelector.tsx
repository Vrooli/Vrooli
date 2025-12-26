import { ChevronDown, Bot } from "lucide-react";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import type { Model } from "../../lib/api";

interface ModelSelectorProps {
  models: Model[];
  selectedModel: string;
  onSelectModel: (modelId: string) => void;
  /** Optional label shown above the selector */
  label?: string;
  /** If true, shows a more compact inline version */
  compact?: boolean;
}

export function ModelSelector({
  models,
  selectedModel,
  onSelectModel,
  label,
  compact = false,
}: ModelSelectorProps) {
  const currentModel = models.find((m) => m.id === selectedModel);

  if (compact) {
    return (
      <Dropdown
        trigger={
          <button
            className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
            data-testid="model-selector-button"
          >
            <Bot className="h-3.5 w-3.5" />
            <span>{currentModel?.name || selectedModel}</span>
            <ChevronDown className="h-3 w-3" />
          </button>
        }
        className="w-72"
      >
        <div className="p-2">
          <p className="text-xs text-slate-500 px-2 mb-1">Select a model</p>
          {models.map((model) => (
            <DropdownItem
              key={model.id}
              onClick={() => onSelectModel(model.id)}
              className={selectedModel === model.id ? "bg-white/10" : ""}
            >
              <div>
                <div className="font-medium">{model.name}</div>
                {model.description && (
                  <div className="text-xs text-slate-500 mt-0.5">{model.description}</div>
                )}
              </div>
            </DropdownItem>
          ))}
        </div>
      </Dropdown>
    );
  }

  return (
    <div>
      {label && <p className="text-xs text-slate-500 mb-2">{label}</p>}
      <Dropdown
        trigger={
          <button
            className="w-full flex items-center justify-between gap-2 px-4 py-3 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-left"
            data-testid="model-selector-button"
          >
            <div className="flex items-center gap-3">
              <Bot className="h-5 w-5 text-indigo-400" />
              <div>
                <div className="font-medium text-white">{currentModel?.name || selectedModel}</div>
                {currentModel?.description && (
                  <div className="text-xs text-slate-500">{currentModel.description}</div>
                )}
              </div>
            </div>
            <ChevronDown className="h-4 w-4 text-slate-400" />
          </button>
        }
        className="w-full"
      >
        <div className="p-2">
          {models.map((model) => (
            <DropdownItem
              key={model.id}
              onClick={() => onSelectModel(model.id)}
              className={selectedModel === model.id ? "bg-white/10" : ""}
            >
              <div className="flex items-center gap-3">
                <Bot className="h-4 w-4 text-slate-400" />
                <div>
                  <div className="font-medium">{model.name}</div>
                  {model.description && (
                    <div className="text-xs text-slate-500 mt-0.5">{model.description}</div>
                  )}
                </div>
              </div>
            </DropdownItem>
          ))}
        </div>
      </Dropdown>
    </div>
  );
}
