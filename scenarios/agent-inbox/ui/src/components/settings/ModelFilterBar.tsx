import { ChevronDown, Building2, ArrowUpDown } from "lucide-react";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import {
  type SortOption,
  type ModalityFilter,
  SORT_OPTIONS,
  MODALITY_OPTIONS,
  formatProviderName,
} from "./modelSelectorUtils";
import { MessageSquare, Type, Image } from "lucide-react";

const MODALITY_ICON_MAP: Record<string, typeof MessageSquare> = {
  MessageSquare,
  Type,
  Image,
};

interface ModelFilterBarProps {
  selectedProvider: string | null;
  onProviderChange: (provider: string | null) => void;
  providers: Array<{ name: string; count: number }>;
  totalModelCount: number;
  modalityFilter: ModalityFilter;
  onModalityChange: (filter: ModalityFilter) => void;
  sortBy: SortOption;
  onSortChange: (sort: SortOption) => void;
}

export function ModelFilterBar({
  selectedProvider,
  onProviderChange,
  providers,
  totalModelCount,
  modalityFilter,
  onModalityChange,
  sortBy,
  onSortChange,
}: ModelFilterBarProps) {
  const selectedProviderLabel = selectedProvider
    ? formatProviderName(selectedProvider)
    : "All providers";

  const selectedSortLabel = SORT_OPTIONS.find((o) => o.value === sortBy)?.label ?? "Sort";
  const selectedModalityOption = MODALITY_OPTIONS.find((o) => o.value === modalityFilter);
  const ModalityIcon = selectedModalityOption ? MODALITY_ICON_MAP[selectedModalityOption.icon] : undefined;

  return (
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
          <DropdownItem onClick={() => onProviderChange(null)} className={!selectedProvider ? "bg-white/10" : ""}>
            <span className="flex-1">All providers</span>
            <span className="text-xs text-slate-500">{totalModelCount}</span>
          </DropdownItem>
          <div className="my-1 border-t border-white/10" />
          {providers.map(({ name, count }) => (
            <DropdownItem key={name} onClick={() => onProviderChange(name)} className={selectedProvider === name ? "bg-white/10" : ""}>
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
            {ModalityIcon && <ModalityIcon className="h-3.5 w-3.5 text-slate-400" />}
            <span className={modalityFilter !== "all" ? "text-white" : "text-slate-400"}>
              {selectedModalityOption?.label}
            </span>
            <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
          </button>
        }
        className="w-48"
      >
        <div className="p-1">
          {MODALITY_OPTIONS.map((option) => {
            const OptIcon = MODALITY_ICON_MAP[option.icon];
            return (
              <DropdownItem key={option.value} onClick={() => onModalityChange(option.value)} className={modalityFilter === option.value ? "bg-white/10" : ""}>
                {OptIcon && <OptIcon className="h-4 w-4 text-slate-400" />}
                <span className="flex-1">{option.label}</span>
              </DropdownItem>
            );
          })}
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
            <DropdownItem key={option.value} onClick={() => onSortChange(option.value)} className={sortBy === option.value ? "bg-white/10" : ""}>
              {option.label}
            </DropdownItem>
          ))}
        </div>
      </Dropdown>
    </div>
  );
}
