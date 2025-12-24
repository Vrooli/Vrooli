/**
 * AssetPicker - Modal to select a brand asset for watermarks/cards
 * Supports both user-uploaded assets and built-in assets (like Vrooli Ascension)
 */

import { useEffect, useState, useCallback, useMemo } from 'react';
import { X, Image as ImageIcon, Check, Upload, ExternalLink, Loader2, Sparkles } from 'lucide-react';
import { useAssetStore } from '@stores/assetStore';
import type { AssetType } from '@lib/storage';
import { getBuiltInAssetsByType, getAllBuiltInAssets } from '@lib/builtInAssets';

interface AssetPickerProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (assetId: string | null) => void;
  selectedId: string | null;
  filterType?: AssetType;
  title?: string;
  /** If true, only show built-in assets (for restricted tiers) */
  builtInOnly?: boolean;
  /** Highlight this asset as recommended */
  recommendedAssetId?: string;
}

export function AssetPicker({
  isOpen,
  onClose,
  onSelect,
  selectedId,
  filterType,
  title = 'Select Asset',
  builtInOnly = false,
  recommendedAssetId,
}: AssetPickerProps) {
  const { assets, isLoading, isInitialized, initialize } = useAssetStore();
  const [localSelected, setLocalSelected] = useState<string | null>(selectedId);

  // Initialize store if needed
  useEffect(() => {
    if (isOpen && !isInitialized) {
      initialize();
    }
  }, [isOpen, isInitialized, initialize]);

  // Reset local selection when modal opens
  useEffect(() => {
    if (isOpen) {
      setLocalSelected(selectedId);
    }
  }, [isOpen, selectedId]);

  // Get built-in assets
  const builtInAssets = useMemo(() => {
    if (filterType) {
      return getBuiltInAssetsByType(filterType);
    }
    return getAllBuiltInAssets();
  }, [filterType]);

  // Filter user assets by type
  const userAssets = useMemo(() => {
    if (builtInOnly) return [];
    return filterType ? assets.filter((a) => a.type === filterType) : assets;
  }, [assets, filterType, builtInOnly]);

  // Combined assets: built-ins first, then user assets
  const allAssets = useMemo(() => {
    return [...builtInAssets, ...userAssets];
  }, [builtInAssets, userAssets]);

  const handleConfirm = useCallback(() => {
    onSelect(localSelected);
    onClose();
  }, [localSelected, onSelect, onClose]);

  const handleClear = useCallback(() => {
    onSelect(null);
    onClose();
  }, [onSelect, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-2xl mx-4 shadow-2xl animate-fade-in flex flex-col max-h-[80vh]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-700">
          <div className="flex items-center gap-3">
            <ImageIcon size={20} className="text-flow-accent" />
            <h3 className="text-lg font-semibold text-white">{title}</h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={24} className="text-gray-400 animate-spin" />
            </div>
          ) : allAssets.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <div className="p-4 bg-gray-800 rounded-full mb-4">
                <Upload size={32} className="text-gray-500" />
              </div>
              <h4 className="text-lg font-medium text-white mb-1">No assets available</h4>
              <p className="text-sm text-gray-500 max-w-xs mb-4">
                {filterType
                  ? `Upload ${filterType} assets in the Branding tab to use them here.`
                  : 'Upload assets in the Branding tab to use them here.'}
              </p>
              <a
                href="#branding"
                className="flex items-center gap-2 text-sm text-flow-accent hover:underline"
                onClick={onClose}
              >
                Go to Branding
                <ExternalLink size={14} />
              </a>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Built-in Assets Section */}
              {builtInAssets.length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <Sparkles size={14} className="text-amber-400" />
                    <span className="text-xs font-medium text-gray-400 uppercase tracking-wide">
                      Built-in Assets
                    </span>
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-4 gap-3">
                    {builtInAssets.map((asset) => {
                      const isSelected = localSelected === asset.id;
                      const isRecommended = recommendedAssetId === asset.id;
                      return (
                        <button
                          key={asset.id}
                          type="button"
                          onClick={() => setLocalSelected(asset.id)}
                          className={`
                            relative aspect-square rounded-lg overflow-hidden border-2 transition-all
                            ${
                              isSelected
                                ? 'border-flow-accent ring-2 ring-flow-accent/50'
                                : isRecommended
                                  ? 'border-amber-500/50 ring-1 ring-amber-500/30'
                                  : 'border-gray-700 hover:border-gray-600'
                            }
                          `}
                        >
                          {asset.url ? (
                            <img
                              src={asset.url}
                              alt={asset.name}
                              className="w-full h-full object-contain bg-gray-800 p-2"
                            />
                          ) : (
                            <div className="w-full h-full bg-gray-800 flex items-center justify-center">
                              <ImageIcon size={24} className="text-gray-600" />
                            </div>
                          )}
                          {isSelected && (
                            <div className="absolute inset-0 bg-flow-accent/20 flex items-center justify-center">
                              <div className="p-2 bg-flow-accent rounded-full">
                                <Check size={16} className="text-white" />
                              </div>
                            </div>
                          )}
                          {isRecommended && !isSelected && (
                            <div className="absolute top-1 right-1">
                              <span className="px-1.5 py-0.5 text-[10px] font-medium bg-amber-500 text-amber-950 rounded">
                                Default
                              </span>
                            </div>
                          )}
                          <div className="absolute bottom-0 left-0 right-0 p-1.5 bg-gradient-to-t from-black/80 to-transparent">
                            <span className="text-xs text-white truncate block">{asset.name}</span>
                            {asset.description && (
                              <span className="text-[10px] text-gray-400 truncate block">
                                {asset.description}
                              </span>
                            )}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* User Assets Section */}
              {userAssets.length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <Upload size={14} className="text-gray-500" />
                    <span className="text-xs font-medium text-gray-400 uppercase tracking-wide">
                      Your Assets
                    </span>
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-4 gap-3">
                    {userAssets.map((asset) => {
                      const isSelected = localSelected === asset.id;
                      return (
                        <button
                          key={asset.id}
                          type="button"
                          onClick={() => setLocalSelected(asset.id)}
                          className={`
                            relative aspect-square rounded-lg overflow-hidden border-2 transition-all
                            ${
                              isSelected
                                ? 'border-flow-accent ring-2 ring-flow-accent/50'
                                : 'border-gray-700 hover:border-gray-600'
                            }
                          `}
                        >
                          {asset.thumbnail ? (
                            <img
                              src={asset.thumbnail}
                              alt={asset.name}
                              className="w-full h-full object-cover"
                            />
                          ) : (
                            <div className="w-full h-full bg-gray-800 flex items-center justify-center">
                              <ImageIcon size={24} className="text-gray-600" />
                            </div>
                          )}
                          {isSelected && (
                            <div className="absolute inset-0 bg-flow-accent/20 flex items-center justify-center">
                              <div className="p-2 bg-flow-accent rounded-full">
                                <Check size={16} className="text-white" />
                              </div>
                            </div>
                          )}
                          <div className="absolute bottom-0 left-0 right-0 p-1.5 bg-gradient-to-t from-black/80 to-transparent">
                            <span className="text-xs text-white truncate block">{asset.name}</span>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Hint for built-in only mode */}
              {builtInOnly && (
                <div className="p-3 rounded-lg bg-amber-950/20 border border-amber-500/20 text-center">
                  <p className="text-xs text-amber-200/80">
                    Upgrade your plan to use custom logos
                  </p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-gray-700 bg-gray-800/50">
          <button
            type="button"
            onClick={handleClear}
            className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
            disabled={!selectedId}
          >
            Clear Selection
          </button>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              className="px-4 py-2 text-sm bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors flex items-center gap-2"
            >
              <Check size={16} />
              Confirm
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default AssetPicker;
