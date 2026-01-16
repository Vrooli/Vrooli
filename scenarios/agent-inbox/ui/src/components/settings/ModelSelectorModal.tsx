import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { Search, Bot, Check, ChevronDown, Building2, ArrowUpDown, Image, MessageSquare, Type, Clock, X, Sparkles } from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Input } from "../ui/input";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import type { Model } from "../../lib/api";
import { supportsImageGeneration } from "../../lib/modelCapabilities";

interface ModelSelectorModalProps {
  open: boolean;
  onClose: () => void;
  models: Model[];
  selectedModel: string;
  onSelectModel: (modelId: string) => void;
}

type SortOption = "name" | "price-asc" | "price-desc" | "context-desc";
type ModalityFilter = "all" | "text" | "image" | "text+image";

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: "name", label: "Name (A-Z)" },
  { value: "price-asc", label: "Price (Low to High)" },
  { value: "price-desc", label: "Price (High to Low)" },
  { value: "context-desc", label: "Context (Largest)" },
];

const MODALITY_OPTIONS: { value: ModalityFilter; label: string; icon: typeof MessageSquare }[] = [
  { value: "all", label: "All modalities", icon: MessageSquare },
  { value: "text", label: "Text only", icon: Type },
  { value: "image", label: "Image support", icon: Image },
  { value: "text+image", label: "Text + Image", icon: Image },
];

const RECENT_MODELS_KEY = "recentModels";
const MAX_RECENT_MODELS = 5;

/**
 * Get recently used model IDs from localStorage.
 */
function getRecentModelIds(): string[] {
  try {
    const stored = localStorage.getItem(RECENT_MODELS_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((id): id is string => typeof id === "string");
  } catch {
    return [];
  }
}

/**
 * Add a model ID to the recent models list.
 * Moves it to the front if already present, maintains max size.
 */
function addRecentModelId(modelId: string): void {
  try {
    const current = getRecentModelIds();
    const filtered = current.filter((id) => id !== modelId);
    const updated = [modelId, ...filtered].slice(0, MAX_RECENT_MODELS);
    localStorage.setItem(RECENT_MODELS_KEY, JSON.stringify(updated));
  } catch {
    // Silently fail if localStorage is unavailable
  }
}

/**
 * Remove a model ID from the recent models list.
 */
function removeRecentModelId(modelId: string): string[] {
  try {
    const current = getRecentModelIds();
    const updated = current.filter((id) => id !== modelId);
    localStorage.setItem(RECENT_MODELS_KEY, JSON.stringify(updated));
    return updated;
  } catch {
    return [];
  }
}

/**
 * Extract provider from model ID or use the provider field.
 */
function getProvider(model: Model): string {
  if (model.provider) {
    return model.provider;
  }
  const parts = model.id.split("/");
  const firstPart = parts[0];
  if (parts.length >= 2 && firstPart) {
    return firstPart;
  }
  return "other";
}

/**
 * Capitalize provider name.
 */
function formatProviderName(provider: string): string {
  return provider
    .split(/[-_]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}

/**
 * Format price per million tokens.
 */
function formatPrice(pricePerToken: number): string {
  const pricePerMillion = pricePerToken * 1_000_000;
  if (pricePerMillion < 0.01) {
    return "<$0.01";
  }
  if (pricePerMillion < 1) {
    return `$${pricePerMillion.toFixed(2)}`;
  }
  return `$${pricePerMillion.toFixed(1)}`;
}

/**
 * Get the combined price for sorting (prompt + completion average).
 */
function getCombinedPrice(model: Model): number {
  if (!model.pricing) return Infinity;
  return (model.pricing.prompt + model.pricing.completion) / 2;
}

/**
 * Check if model supports a specific input modality.
 */
function supportsModality(model: Model, modality: string): boolean {
  return model.architecture?.input?.includes(modality) ?? false;
}

/**
 * Get modality badges for a model.
 */
function getModalityBadges(model: Model): string[] {
  return model.architecture?.input ?? ["text"];
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
  const [sortBy, setSortBy] = useState<SortOption>("name");
  const [modalityFilter, setModalityFilter] = useState<ModalityFilter>("all");
  const [recentModelIds, setRecentModelIds] = useState<string[]>([]);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Reset state and load recent models when modal opens
  useEffect(() => {
    if (open) {
      setSearchQuery("");
      setSelectedProvider(null);
      setSortBy("name");
      setModalityFilter("all");
      setRecentModelIds(getRecentModelIds());
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
    return Array.from(providerMap.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([name, count]) => ({ name, count }));
  }, [models]);

  // Get recent models that exist in the current models list
  const recentModels = useMemo(() => {
    const modelMap = new Map(models.map((m) => [m.id, m]));
    return recentModelIds
      .map((id) => modelMap.get(id))
      .filter((m): m is Model => m !== undefined);
  }, [models, recentModelIds]);

  // Filter and sort models
  const filteredModels = useMemo(() => {
    let result = models;

    // Filter by provider
    if (selectedProvider) {
      result = result.filter((model) => getProvider(model) === selectedProvider);
    }

    // Filter by modality
    if (modalityFilter !== "all") {
      result = result.filter((model) => {
        if (modalityFilter === "text") {
          return supportsModality(model, "text") && !supportsModality(model, "image");
        }
        if (modalityFilter === "image") {
          return supportsModality(model, "image");
        }
        if (modalityFilter === "text+image") {
          return supportsModality(model, "text") && supportsModality(model, "image");
        }
        return true;
      });
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

    // Sort
    result = [...result].sort((a, b) => {
      switch (sortBy) {
        case "name":
          return a.name.localeCompare(b.name);
        case "price-asc":
          return getCombinedPrice(a) - getCombinedPrice(b);
        case "price-desc":
          return getCombinedPrice(b) - getCombinedPrice(a);
        case "context-desc":
          return (b.context_length ?? 0) - (a.context_length ?? 0);
        default:
          return 0;
      }
    });

    return result;
  }, [models, searchQuery, selectedProvider, sortBy, modalityFilter]);

  // Group filtered models by provider (only when sorting by name)
  const groupedModels = useMemo(() => {
    if (sortBy !== "name") {
      // Don't group when sorting by something other than name
      return [["", filteredModels] as [string, Model[]]];
    }
    const groups = new Map<string, Model[]>();
    for (const model of filteredModels) {
      const provider = getProvider(model);
      const providerModels = groups.get(provider) ?? [];
      if (!groups.has(provider)) {
        groups.set(provider, providerModels);
      }
      providerModels.push(model);
    }
    return Array.from(groups.entries()).sort((a, b) => a[0].localeCompare(b[0]));
  }, [filteredModels, sortBy]);

  // Show recent section only when in default view (no search/filters active)
  const showRecentSection =
    recentModels.length > 0 &&
    !searchQuery.trim() &&
    !selectedProvider &&
    modalityFilter === "all" &&
    sortBy === "name";

  // Create a set of recent model IDs for quick lookup (to avoid showing duplicates in main list)
  const recentModelIdSet = useMemo(
    () => new Set(showRecentSection ? recentModelIds : []),
    [showRecentSection, recentModelIds]
  );

  const handleSelectModel = useCallback((modelId: string) => {
    addRecentModelId(modelId);
    onSelectModel(modelId);
    onClose();
  }, [onSelectModel, onClose]);

  const handleRemoveRecentModel = useCallback((e: React.MouseEvent, modelId: string) => {
    e.stopPropagation(); // Prevent selecting the model when clicking remove
    const updated = removeRecentModelId(modelId);
    setRecentModelIds(updated);
  }, []);

  const selectedProviderLabel = selectedProvider
    ? formatProviderName(selectedProvider)
    : "All providers";

  const selectedSortLabel = SORT_OPTIONS.find((o) => o.value === sortBy)?.label ?? "Sort";
  const selectedModalityOption = MODALITY_OPTIONS.find((o) => o.value === modalityFilter);

  return (
    <Dialog open={open} onClose={onClose} className="max-w-2xl">
      <DialogHeader onClose={onClose}>Select Model</DialogHeader>
      <DialogBody className="p-0">
        {/* Search bar */}
        <div className="p-4 border-b border-white/10">
          <div className="relative">
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
        </div>

        {/* Filters row */}
        <div className="px-4 py-3 border-b border-white/10 flex flex-wrap gap-2">
          {/* Provider dropdown */}
          <Dropdown
            trigger={
              <button
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-sm"
                data-testid="provider-filter-button"
              >
                <Building2 className="h-3.5 w-3.5 text-slate-400" />
                <span className={selectedProvider ? "text-white" : "text-slate-400"}>
                  {selectedProviderLabel}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
              </button>
            }
            className="w-56 max-h-80 overflow-y-auto"
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

          {/* Modality dropdown */}
          <Dropdown
            trigger={
              <button
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-sm"
                data-testid="modality-filter-button"
              >
                {selectedModalityOption && <selectedModalityOption.icon className="h-3.5 w-3.5 text-slate-400" />}
                <span className={modalityFilter !== "all" ? "text-white" : "text-slate-400"}>
                  {selectedModalityOption?.label}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
              </button>
            }
            className="w-48"
          >
            <div className="p-1">
              {MODALITY_OPTIONS.map((option) => (
                <DropdownItem
                  key={option.value}
                  onClick={() => setModalityFilter(option.value)}
                  className={modalityFilter === option.value ? "bg-white/10" : ""}
                >
                  <option.icon className="h-4 w-4 text-slate-400" />
                  <span className="flex-1">{option.label}</span>
                </DropdownItem>
              ))}
            </div>
          </Dropdown>

          {/* Sort dropdown */}
          <Dropdown
            trigger={
              <button
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 hover:bg-white/10 hover:border-white/20 transition-colors text-sm"
                data-testid="sort-button"
              >
                <ArrowUpDown className="h-3.5 w-3.5 text-slate-400" />
                <span className={sortBy !== "name" ? "text-white" : "text-slate-400"}>
                  {selectedSortLabel}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
              </button>
            }
            className="w-48"
          >
            <div className="p-1">
              {SORT_OPTIONS.map((option) => (
                <DropdownItem
                  key={option.value}
                  onClick={() => setSortBy(option.value)}
                  className={sortBy === option.value ? "bg-white/10" : ""}
                >
                  {option.label}
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
              {/* Recent models section */}
              {showRecentSection && (
                <div className="mb-2">
                  <div className="px-3 py-2 flex items-center gap-2">
                    <Clock className="h-3.5 w-3.5 text-slate-500" />
                    <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Recent
                    </h3>
                  </div>
                  {recentModels.map((model) => {
                    const isSelected = model.id === selectedModel;
                    const modalities = getModalityBadges(model);
                    const hasImageSupport = modalities.includes("image");

                    const canGenerateImages = supportsImageGeneration(model);

                    return (
                      <button
                        key={`recent-${model.id}`}
                        onClick={() => handleSelectModel(model.id)}
                        className={`w-full flex items-start gap-3 p-3 rounded-lg text-left transition-colors ${
                          isSelected
                            ? "bg-indigo-500/20 border border-indigo-500/50"
                            : "hover:bg-white/5 border border-transparent"
                        }`}
                        data-testid={`recent-model-option-${model.id}`}
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
                                <span className="text-slate-600">•</span>
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
                        </div>
                        <div
                          role="button"
                          tabIndex={0}
                          onClick={(e) => handleRemoveRecentModel(e, model.id)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.preventDefault();
                              handleRemoveRecentModel(e as unknown as React.MouseEvent, model.id);
                            }
                          }}
                          className="shrink-0 p-1.5 rounded-md hover:bg-white/10 text-slate-500 hover:text-slate-300 transition-colors"
                          title="Remove from recent"
                          data-testid={`remove-recent-${model.id}`}
                        >
                          <X className="h-4 w-4" />
                        </div>
                      </button>
                    );
                  })}
                  <div className="my-2 mx-3 border-t border-white/10" />
                </div>
              )}
              {groupedModels.map(([provider, providerModels]) => {
                // Filter out models already shown in Recent section
                const displayModels = showRecentSection
                  ? providerModels.filter((m) => !recentModelIdSet.has(m.id))
                  : providerModels;

                if (displayModels.length === 0) return null;

                return (
                <div key={provider || "all"}>
                  {/* Provider section header (only show when sorting by name and not filtering by provider) */}
                  {provider && !selectedProvider && sortBy === "name" && (
                    <div className="px-3 py-2 mt-2 first:mt-0">
                      <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                        {formatProviderName(provider)}
                      </h3>
                    </div>
                  )}
                  {displayModels.map((model) => {
                    const isSelected = model.id === selectedModel;
                    const modalities = getModalityBadges(model);
                    const hasImageSupport = modalities.includes("image");
                    const canGenerateImages = supportsImageGeneration(model);

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

                          {/* Metadata row: modality badges, context, pricing */}
                          <div className="text-xs text-slate-500 mt-1 flex items-center gap-2 flex-wrap">
                            {/* Modality badges */}
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

                            {/* Context length */}
                            {model.context_length && (
                              <span>{(model.context_length / 1000).toFixed(0)}K ctx</span>
                            )}

                            {/* Pricing */}
                            {model.pricing && (
                              <>
                                <span className="text-slate-600">•</span>
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
                );
              })}
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
