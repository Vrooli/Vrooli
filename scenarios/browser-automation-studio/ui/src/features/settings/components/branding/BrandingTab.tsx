/**
 * BrandingTab - Main content for the Branding settings tab
 *
 * Manages brand asset library - upload, view, organize, and delete assets.
 */

import { useEffect } from 'react';
import { Image as ImageIcon, HardDrive, Info } from 'lucide-react';
import { useAssetStore } from '@stores/assetStore';
import { AssetUploader } from './AssetUploader';
import { AssetGallery } from './AssetGallery';

export function BrandingTab() {
  const { initialize, isInitialized, storageInfo, error, clearError } = useAssetStore();

  // Initialize asset store on mount
  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [initialize, isInitialized]);

  // Format storage size
  const formatStorage = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <ImageIcon size={24} className="text-pink-400" />
        <div>
          <h2 className="text-lg font-semibold text-white">Brand Assets</h2>
          <p className="text-sm text-gray-400">
            Upload logos and backgrounds for watermarks, intro, and outro cards
          </p>
        </div>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="flex items-start gap-3 p-4 bg-red-500/10 border border-red-500/30 rounded-lg">
          <Info size={18} className="text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-red-300">{error}</p>
          </div>
          <button
            type="button"
            onClick={clearError}
            className="text-xs text-red-400 hover:text-red-300"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Storage Info */}
      {storageInfo && (
        <div className="flex items-center gap-3 p-3 bg-gray-800/50 rounded-lg">
          <HardDrive size={18} className="text-gray-400" />
          <div className="flex-1">
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-400">
                {storageInfo.count} asset{storageInfo.count !== 1 ? 's' : ''} stored
              </span>
              <span className="text-gray-500">{formatStorage(storageInfo.used)} used</span>
            </div>
            {storageInfo.total > 0 && (
              <div className="mt-2">
                <div className="h-1.5 bg-gray-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-flow-accent rounded-full transition-all"
                    style={{
                      width: `${Math.min(100, (storageInfo.used / storageInfo.total) * 100)}%`,
                    }}
                  />
                </div>
                <div className="flex justify-between mt-1 text-xs text-gray-600">
                  <span>0</span>
                  <span>{formatStorage(storageInfo.total)}</span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Info Box */}
      <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-lg">
        <div className="flex items-start gap-3">
          <Info size={18} className="text-blue-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-blue-200 font-medium">How to use brand assets</p>
            <p className="text-xs text-blue-300/80 mt-1">
              Upload your logos and background images here, then use them in the{' '}
              <span className="font-medium">Replay</span> tab to add watermarks, intro cards, and
              outro cards to your workflow replays.
            </p>
          </div>
        </div>
      </div>

      {/* Upload Section */}
      <div className="border border-gray-700 rounded-lg overflow-hidden">
        <div className="px-4 py-3 bg-gray-800/50 border-b border-gray-700">
          <h3 className="text-sm font-medium text-white">Upload Assets</h3>
        </div>
        <div className="p-4">
          <AssetUploader />
        </div>
      </div>

      {/* Asset Library */}
      <div className="border border-gray-700 rounded-lg overflow-hidden">
        <div className="px-4 py-3 bg-gray-800/50 border-b border-gray-700">
          <h3 className="text-sm font-medium text-white">Asset Library</h3>
        </div>
        <div className="p-4">
          <AssetGallery />
        </div>
      </div>
    </div>
  );
}

export default BrandingTab;
