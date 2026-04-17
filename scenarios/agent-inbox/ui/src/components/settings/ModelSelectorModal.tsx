import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { Search, Bot, Clock } from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Input } from "../ui/input";
import type { Model } from "../../lib/api";
import { ModelCard } from "./ModelCard";
import { ModelFilterBar } from "./ModelFilterBar";
import {
  type SortOption,
  type ModalityFilter,
  getRecentModelIds,
  addRecentModelId,
  removeRecentModelId,
  getProvider,
  formatProviderName,
  getCombinedPrice,
  supportsModality,
} from "./modelSelectorUtils";

interface ModelSelectorModalProps {
  open: boolean;
  onClose: () => void;
  models: Model[];
  selectedModel: string;
  onSelectModel: (modelId: string) => void;
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

  useEffect(() => {
    if (open) {
      setSearchQuery("");
      setSelectedProvider(null);
      setSortBy("name");
      setModalityFilter("all");
      setRecentModelIds(getRecentModelIds());
      const timer = setTimeout(() => searchInputRef.current?.focus(), 50);
      return () => clearTimeout(timer);
    }
  }, [open]);

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

  const recentModels = useMemo(() => {
    const modelMap = new Map(models.map((m) => [m.id, m]));
    return recentModelIds
      .map((id) => modelMap.get(id))
      .filter((m): m is Model => m !== undefined);
  }, [models, recentModelIds]);

  const filteredModels = useMemo(() => {
    let result = models;
    if (selectedProvider) result = result.filter((m) => getProvider(m) === selectedProvider);
    if (modalityFilter !== "all") {
      result = result.filter((model) => {
        if (modalityFilter === "text") return supportsModality(model, "text") && !supportsModality(model, "image");
        if (modalityFilter === "image") return supportsModality(model, "image");
        return supportsModality(model, "text") && supportsModality(model, "image");
      });
    }
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((m) => m.name.toLowerCase().includes(query) || m.id.toLowerCase().includes(query) || m.description?.toLowerCase().includes(query));
    }
    return [...result].sort((a, b) => {
      switch (sortBy) {
        case "name": return a.name.localeCompare(b.name);
        case "price-asc": return getCombinedPrice(a) - getCombinedPrice(b);
        case "price-desc": return getCombinedPrice(b) - getCombinedPrice(a);
        case "context-desc": return (b.context_length ?? 0) - (a.context_length ?? 0);
        default: return 0;
      }
    });
  }, [models, searchQuery, selectedProvider, sortBy, modalityFilter]);

  const groupedModels = useMemo(() => {
    if (sortBy !== "name") return [["", filteredModels] as [string, Model[]]];
    const groups = new Map<string, Model[]>();
    for (const model of filteredModels) {
      const provider = getProvider(model);
      const existing = groups.get(provider);
      if (existing) {
        existing.push(model);
      } else {
        groups.set(provider, [model]);
      }
    }
    return Array.from(groups.entries()).sort((a, b) => a[0].localeCompare(b[0]));
  }, [filteredModels, sortBy]);

  const showRecentSection = recentModels.length > 0 && !searchQuery.trim() && !selectedProvider && modalityFilter === "all" && sortBy === "name";
  const recentModelIdSet = useMemo(() => new Set(showRecentSection ? recentModelIds : []), [showRecentSection, recentModelIds]);

  const handleSelectModel = useCallback((modelId: string) => {
    addRecentModelId(modelId);
    onSelectModel(modelId);
    onClose();
  }, [onSelectModel, onClose]);

  const handleRemoveRecentModel = useCallback((e: React.MouseEvent, modelId: string) => {
    e.stopPropagation();
    setRecentModelIds(removeRecentModelId(modelId));
  }, []);

  return (
    <Dialog open={open} onClose={onClose} className="max-w-2xl">
      <DialogHeader onClose={onClose}>Select Model</DialogHeader>
      <DialogBody className="p-0">
        <div className="p-4 border-b border-white/10">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
            <Input ref={searchInputRef} value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} placeholder="Search models..." className="pl-10" data-testid="model-search-input" />
          </div>
        </div>

        <ModelFilterBar
          selectedProvider={selectedProvider}
          onProviderChange={setSelectedProvider}
          providers={providers}
          totalModelCount={models.length}
          modalityFilter={modalityFilter}
          onModalityChange={setModalityFilter}
          sortBy={sortBy}
          onSortChange={setSortBy}
        />

        <div className="max-h-[400px] overflow-y-auto">
          {filteredModels.length === 0 ? (
            <div className="p-8 text-center text-slate-500">
              <Bot className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>No models found</p>
              {searchQuery && <p className="text-sm mt-1">Try a different search term</p>}
            </div>
          ) : (
            <div className="p-2">
              {showRecentSection && (
                <div className="mb-2">
                  <div className="px-3 py-2 flex items-center gap-2">
                    <Clock className="h-3.5 w-3.5 text-slate-500" />
                    <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Recent</h3>
                  </div>
                  {recentModels.map((model) => (
                    <ModelCard key={`recent-${model.id}`} model={model} isSelected={model.id === selectedModel} onSelect={handleSelectModel} onRemove={handleRemoveRecentModel} testIdPrefix="recent-model-option" />
                  ))}
                  <div className="my-2 mx-3 border-t border-white/10" />
                </div>
              )}
              {groupedModels.map(([provider, providerModels]) => {
                const displayModels = showRecentSection ? providerModels.filter((m) => !recentModelIdSet.has(m.id)) : providerModels;
                if (displayModels.length === 0) return null;
                return (
                  <div key={provider || "all"}>
                    {provider && !selectedProvider && sortBy === "name" && (
                      <div className="px-3 py-2 mt-2 first:mt-0">
                        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">{formatProviderName(provider)}</h3>
                      </div>
                    )}
                    {displayModels.map((model) => (
                      <ModelCard key={model.id} model={model} isSelected={model.id === selectedModel} onSelect={handleSelectModel} />
                    ))}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="p-3 border-t border-white/10 text-center text-xs text-slate-500">
          {filteredModels.length === models.length ? `${models.length} models available` : `${filteredModels.length} of ${models.length} models`}
        </div>
      </DialogBody>
    </Dialog>
  );
}
