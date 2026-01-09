/**
 * AssetGallery - Grid view of uploaded brand assets
 */

import { useState, useCallback, useMemo } from 'react';
import {
  Image as ImageIcon,
  Trash2,
  Edit2,
  Check,
  X,
  Tag,
  Loader2,
  FolderOpen,
} from 'lucide-react';
import { useAssetStore } from '@stores/assetStore';
import type { AssetType, BrandAsset } from '@lib/storage';

interface AssetGalleryProps {
  filterType?: AssetType | 'all';
  onAssetSelect?: (asset: BrandAsset) => void;
  selectable?: boolean;
  selectedId?: string | null;
}

type TypeFilter = AssetType | 'all';

const TYPE_OPTIONS: Array<{ id: TypeFilter; label: string }> = [
  { id: 'all', label: 'All' },
  { id: 'logo', label: 'Logos' },
  { id: 'background', label: 'Backgrounds' },
  { id: 'other', label: 'Other' },
];

export function AssetGallery({
  filterType: propFilterType,
  onAssetSelect,
  selectable = false,
  selectedId,
}: AssetGalleryProps) {
  const { assets, isLoading, deleteAsset, renameAsset, updateAssetType } = useAssetStore();
  const [activeFilter, setActiveFilter] = useState<TypeFilter>(propFilterType || 'all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [changingTypeId, setChangingTypeId] = useState<string | null>(null);

  // Filter assets
  const filteredAssets = useMemo(() => {
    const filter = propFilterType || activeFilter;
    if (filter === 'all') return assets;
    return assets.filter((a) => a.type === filter);
  }, [assets, activeFilter, propFilterType]);

  // Start editing
  const startEdit = useCallback((asset: BrandAsset) => {
    setEditingId(asset.id);
    setEditName(asset.name);
  }, []);

  // Save edit
  const saveEdit = useCallback(async () => {
    if (!editingId || !editName.trim()) return;
    try {
      await renameAsset(editingId, editName.trim());
    } finally {
      setEditingId(null);
      setEditName('');
    }
  }, [editingId, editName, renameAsset]);

  // Cancel edit
  const cancelEdit = useCallback(() => {
    setEditingId(null);
    setEditName('');
  }, []);

  // Handle delete
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteAsset(id);
      } finally {
        setDeletingId(null);
      }
    },
    [deleteAsset],
  );

  // Handle type change
  const handleTypeChange = useCallback(
    async (id: string, type: AssetType) => {
      try {
        await updateAssetType(id, type);
      } finally {
        setChangingTypeId(null);
      }
    },
    [updateAssetType],
  );

  // Format file size
  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 size={24} className="text-gray-400 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Filter Tabs - only show if no prop filter */}
      {!propFilterType && (
        <div className="flex gap-1 p-1 bg-gray-800/50 rounded-lg">
          {TYPE_OPTIONS.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setActiveFilter(option.id)}
              className={`flex-1 px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                activeFilter === option.id
                  ? 'bg-gray-700 text-surface'
                  : 'text-gray-400 hover:text-surface'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}

      {/* Empty State */}
      {filteredAssets.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <div className="p-4 bg-gray-800 rounded-full mb-4">
            <FolderOpen size={32} className="text-gray-500" />
          </div>
          <h3 className="text-lg font-medium text-surface mb-1">No assets yet</h3>
          <p className="text-sm text-gray-500 max-w-xs">
            Upload logos and background images to use in your replays.
          </p>
        </div>
      )}

      {/* Asset Grid */}
      {filteredAssets.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {filteredAssets.map((asset) => {
            const isEditing = editingId === asset.id;
            const isDeleting = deletingId === asset.id;
            const isChangingType = changingTypeId === asset.id;
            const isSelected = selectable && selectedId === asset.id;

            return (
              <div
                key={asset.id}
                className={`group relative bg-gray-800 rounded-lg overflow-hidden border transition-all ${
                  isSelected
                    ? 'border-flow-accent ring-2 ring-flow-accent/50'
                    : 'border-gray-700 hover:border-gray-600'
                } ${selectable ? 'cursor-pointer' : ''}`}
                onClick={() => selectable && onAssetSelect?.(asset)}
              >
                {/* Thumbnail */}
                <div className="aspect-square bg-gray-900 flex items-center justify-center overflow-hidden">
                  {asset.thumbnail ? (
                    <img
                      src={asset.thumbnail}
                      alt={asset.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <ImageIcon size={32} className="text-gray-600" />
                  )}
                </div>

                {/* Selection Check */}
                {isSelected && (
                  <div className="absolute top-2 right-2 p-1 bg-flow-accent rounded-full">
                    <Check size={14} className="text-white" />
                  </div>
                )}

                {/* Info & Actions */}
                <div className="p-3">
                  {isEditing ? (
                    <div className="flex items-center gap-1">
                      <input
                        type="text"
                        value={editName}
                        onChange={(e) => setEditName(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') saveEdit();
                          if (e.key === 'Escape') cancelEdit();
                        }}
                        className="flex-1 px-2 py-1 text-sm bg-gray-900 border border-gray-600 rounded text-surface"
                        autoFocus
                        onClick={(e) => e.stopPropagation()}
                      />
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          saveEdit();
                        }}
                        className="p-1 text-green-400 hover:bg-green-900/30 rounded"
                      >
                        <Check size={14} />
                      </button>
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          cancelEdit();
                        }}
                        className="p-1 text-gray-400 hover:bg-gray-700 rounded"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ) : (
                    <>
                      <h4 className="text-sm font-medium text-surface truncate" title={asset.name}>
                        {asset.name}
                      </h4>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-xs text-gray-500">{formatSize(asset.sizeBytes)}</span>
                        <span className="text-xs px-1.5 py-0.5 bg-gray-700 text-gray-400 rounded">
                          {asset.type}
                        </span>
                      </div>
                    </>
                  )}
                </div>

                {/* Hover Actions */}
                {!isEditing && !selectable && (
                  <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                    {/* Rename */}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        startEdit(asset);
                      }}
                      className="p-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-surface transition-colors"
                      title="Rename"
                    >
                      <Edit2 size={16} />
                    </button>

                    {/* Change Type */}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setChangingTypeId(asset.id);
                      }}
                      className="p-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-surface transition-colors"
                      title="Change type"
                    >
                      <Tag size={16} />
                    </button>

                    {/* Delete */}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeletingId(asset.id);
                      }}
                      className="p-2 bg-gray-800 hover:bg-red-900/50 rounded-lg text-red-400 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                )}

                {/* Type Change Popup */}
                {isChangingType && (
                  <div
                    className="absolute inset-0 bg-black/80 flex flex-col items-center justify-center gap-2 p-4"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <span className="text-xs text-gray-400 mb-1">Change type to:</span>
                    {(['logo', 'background', 'other'] as AssetType[]).map((type) => (
                      <button
                        key={type}
                        type="button"
                        onClick={() => handleTypeChange(asset.id, type)}
                        className={`w-full px-3 py-2 text-sm rounded-lg transition-colors ${
                          asset.type === type
                            ? 'bg-flow-accent text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        }`}
                      >
                        {type.charAt(0).toUpperCase() + type.slice(1)}
                      </button>
                    ))}
                    <button
                      type="button"
                      onClick={() => setChangingTypeId(null)}
                      className="mt-2 text-xs text-gray-500 hover:text-surface"
                    >
                      Cancel
                    </button>
                  </div>
                )}

                {/* Delete Confirmation */}
                {isDeleting && (
                  <div
                    className="absolute inset-0 bg-black/80 flex flex-col items-center justify-center gap-3 p-4"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <span className="text-sm text-surface text-center">Delete this asset?</span>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setDeletingId(null)}
                        className="px-3 py-1.5 text-sm bg-gray-700 text-surface rounded-lg hover:bg-gray-600"
                      >
                        Cancel
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(asset.id)}
                        className="px-3 py-1.5 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default AssetGallery;
