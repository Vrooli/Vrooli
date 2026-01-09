import { useState } from "react";
import { ChevronDown, Bot } from "lucide-react";
import { ModelSelectorModal } from "./ModelSelectorModal";
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
  const [isModalOpen, setIsModalOpen] = useState(false);
  const currentModel = models.find((m) => m.id === selectedModel);

  const handleOpenModal = () => setIsModalOpen(true);
  const handleCloseModal = () => setIsModalOpen(false);

  if (compact) {
    return (
      <>
        <button
          onClick={handleOpenModal}
          className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
          data-testid="model-selector-button"
        >
          <Bot className="h-3.5 w-3.5" />
          <span>{currentModel?.name || selectedModel}</span>
          <ChevronDown className="h-3 w-3" />
        </button>
        <ModelSelectorModal
          open={isModalOpen}
          onClose={handleCloseModal}
          models={models}
          selectedModel={selectedModel}
          onSelectModel={onSelectModel}
        />
      </>
    );
  }

  return (
    <>
      <div>
        {label && <p className="text-xs text-slate-500 mb-2">{label}</p>}
        <button
          onClick={handleOpenModal}
          className="w-full flex items-center justify-between gap-2 px-4 py-3 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-left"
          data-testid="model-selector-button"
        >
          <div className="flex items-center gap-3">
            <Bot className="h-5 w-5 text-indigo-400" />
            <div>
              <div className="font-medium text-white">{currentModel?.name || selectedModel}</div>
              {currentModel?.description && (
                <div className="text-xs text-slate-500 line-clamp-1">{currentModel.description}</div>
              )}
            </div>
          </div>
          <ChevronDown className="h-4 w-4 text-slate-400" />
        </button>
      </div>
      <ModelSelectorModal
        open={isModalOpen}
        onClose={handleCloseModal}
        models={models}
        selectedModel={selectedModel}
        onSelectModel={onSelectModel}
      />
    </>
  );
}
