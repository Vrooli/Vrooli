/**
 * AssetPicker - Modal to select a brand asset for watermarks/cards
 */

import { useEffect, useState, useCallback } from 'react';
import { X, Image as ImageIcon, Check, Upload, ExternalLink, Loader2 } from 'lucide-react';
import { useAssetStore } from '@stores/assetStore';
import type { AssetType } from '@lib/storage';

interface AssetPickerProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (assetId: string | null) => void;
  selectedId: string | null;
  filterType?: AssetType;
  title?: string;
}

export function AssetPicker({
  isOpen,
  onClose,
  onSelect,
  selectedId,
  filterType,
  title = 'Select Asset',
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

  // Filter assets by type
  const filteredAssets = filterType ? assets.filter((a) => a.type === filterType) : assets;

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
          ) : filteredAssets.length === 0 ? (
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
            <div className="grid grid-cols-3 sm:grid-cols-4 gap-3">
              {filteredAssets.map((asset) => {
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
