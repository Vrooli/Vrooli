import { useState, useMemo, useEffect, useRef } from "react";
import { Search, Bot, Check, ChevronDown, Building2 } from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Input } from "../ui/input";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import type { Model } from "../../lib/api";

interface ModelSelectorModalProps {
  open: boolean;
  onClose: () => void;
  models: Model[];
  selectedModel: string;
  onSelectModel: (modelId: string) => void;
}

/**
 * Extract provider from model ID or use the provider field.
 * Model IDs follow the pattern: provider/model-name (e.g., "anthropic/claude-3.5-sonnet")
 */
function getProvider(model: Model): string {
  if (model.provider) {
    return model.provider;
  }
  // Extract from ID (e.g., "anthropic/claude-3.5-sonnet" -> "anthropic")
  const parts = model.id.split("/");
  if (parts.length >= 2) {
    return parts[0];
  }
  return "other";
}

/**
 * Capitalize the first letter of each word in a provider name.
 * e.g., "anthropic" -> "Anthropic", "openai" -> "Openai"
 */
function formatProviderName(provider: string): string {
  return provider
    .split(/[-_]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}

export function ModelSelectorModal({
  open,
  onClose,
  models,
  selectedModel,
  onSelectModel,
}: ModelSelectorModalProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setSearchQuery("");
      setSelectedProvider(null);
      const timer = setTimeout(() => {
        searchInputRef.current?.focus();
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [open]);

  // Get unique providers and their model counts
  const providers = useMemo(() => {
    const providerMap = new Map<string, number>();
    for (const model of models) {
      const provider = getProvider(model);
      providerMap.set(provider, (providerMap.get(provider) || 0) + 1);
    }
    // Sort alphabetically
    return Array.from(providerMap.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([name, count]) => ({ name, count }));
  }, [models]);

  // Filter models based on search query and selected provider
  const filteredModels = useMemo(() => {
    let result = models;

    // Filter by provider
    if (selectedProvider) {
      result = result.filter((model) => getProvider(model) === selectedProvider);
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (model) =>
          model.name.toLowerCase().includes(query) ||
          model.id.toLowerCase().includes(query) ||
          model.description?.toLowerCase().includes(query)
      );
    }

    return result;
  }, [models, searchQuery, selectedProvider]);

  // Group filtered models by provider
  const groupedModels = useMemo(() => {
    const groups = new Map<string, Model[]>();
    for (const model of filteredModels) {
      const provider = getProvider(model);
      if (!groups.has(provider)) {
        groups.set(provider, []);
      }
      groups.get(provider)!.push(model);
    }
    // Sort groups by provider name
    return Array.from(groups.entries()).sort((a, b) => a[0].localeCompare(b[0]));
  }, [filteredModels]);

  const handleSelectModel = (modelId: string) => {
    onSelectModel(modelId);
    onClose();
  };

  const selectedProviderLabel = selectedProvider
    ? formatProviderName(selectedProvider)
    : "All providers";

  return (
    <Dialog open={open} onClose={onClose} className="max-w-2xl">
      <DialogHeader onClose={onClose}>Select Model</DialogHeader>
      <DialogBody className="p-0">
        {/* Search bar and provider filter */}
        <div className="p-4 border-b border-white/10 flex gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
            <Input
              ref={searchInputRef}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search models..."
              className="pl-10"
              data-testid="model-search-input"
            />
          </div>

          {/* Provider dropdown */}
          <Dropdown
            trigger={
              <button
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-sm whitespace-nowrap"
                data-testid="provider-filter-button"
              >
                <Building2 className="h-4 w-4 text-slate-400" />
                <span className={selectedProvider ? "text-white" : "text-slate-400"}>
                  {selectedProviderLabel}
                </span>
                <ChevronDown className="h-4 w-4 text-slate-400" />
              </button>
            }
            className="w-56 max-h-80 overflow-y-auto"
            align="right"
          >
            <div className="p-1">
              <DropdownItem
                onClick={() => setSelectedProvider(null)}
                className={!selectedProvider ? "bg-white/10" : ""}
              >
                <span className="flex-1">All providers</span>
                <span className="text-xs text-slate-500">{models.length}</span>
              </DropdownItem>
              <div className="my-1 border-t border-white/10" />
              {providers.map(({ name, count }) => (
                <DropdownItem
                  key={name}
                  onClick={() => setSelectedProvider(name)}
                  className={selectedProvider === name ? "bg-white/10" : ""}
                >
                  <span className="flex-1">{formatProviderName(name)}</span>
                  <span className="text-xs text-slate-500">{count}</span>
                </DropdownItem>
              ))}
            </div>
          </Dropdown>
        </div>

        {/* Models list */}
        <div className="max-h-[400px] overflow-y-auto">
          {filteredModels.length === 0 ? (
            <div className="p-8 text-center text-slate-500">
              <Bot className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>No models found</p>
              {searchQuery && (
                <p className="text-sm mt-1">Try a different search term</p>
              )}
            </div>
          ) : (
            <div className="p-2">
              {groupedModels.map(([provider, providerModels]) => (
                <div key={provider}>
                  {/* Provider section header (only show when not filtering by provider) */}
                  {!selectedProvider && (
                    <div className="px-3 py-2 mt-2 first:mt-0">
                      <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        {formatProviderName(provider)}
                      </h3>
                    </div>
                  )}
                  {providerModels.map((model) => {
                    const isSelected = model.id === selectedModel;
                    return (
                      <button
                        key={model.id}
                        onClick={() => handleSelectModel(model.id)}
                        className={`w-full flex items-start gap-3 p-3 rounded-lg text-left transition-colors ${
                          isSelected
                            ? "bg-indigo-500/20 border border-indigo-500/50"
                            : "hover:bg-white/5 border border-transparent"
                        }`}
                        data-testid={`model-option-${model.id}`}
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
                          <div className="text-xs text-slate-500 mt-0.5 flex items-center gap-2">
                            <span>{model.id}</span>
                            {model.context_length && (
                              <>
                                <span className="text-slate-600">•</span>
                                <span>{(model.context_length / 1000).toFixed(0)}K context</span>
                              </>
                            )}
                          </div>
                          {model.description && (
                            <p className="text-sm text-slate-400 mt-1.5 line-clamp-3">
                              {model.description}
                            </p>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer with count */}
        <div className="p-3 border-t border-white/10 text-center text-xs text-slate-500">
          {filteredModels.length === models.length
            ? `${models.length} models available`
            : `${filteredModels.length} of ${models.length} models`}
        </div>
      </DialogBody>
    </Dialog>
  );
}
