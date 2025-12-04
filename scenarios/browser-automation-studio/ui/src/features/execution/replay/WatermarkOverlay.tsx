/**
 * WatermarkOverlay - Renders watermark on replay player
 */

import { useEffect, useState } from 'react';
import type { WatermarkSettings, WatermarkPosition } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';

interface WatermarkOverlayProps {
  settings: WatermarkSettings;
}

// Position CSS classes
const POSITION_CLASSES: Record<WatermarkPosition, string> = {
  'top-left': 'top-0 left-0',
  'top-right': 'top-0 right-0',
  'bottom-left': 'bottom-0 left-0',
  'bottom-right': 'bottom-0 right-0',
  center: 'top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2',
};

export function WatermarkOverlay({ settings }: WatermarkOverlayProps) {
  const { getAssetUrl, getAssetById } = useAssetStore();
  const [imageUrl, setImageUrl] = useState<string | null>(null);

  // Load image URL when asset ID changes
  useEffect(() => {
    let mounted = true;

    const loadUrl = async () => {
      if (!settings.assetId) {
        setImageUrl(null);
        return;
      }

      // First try to use thumbnail for quick display
      const asset = getAssetById(settings.assetId);
      if (asset?.thumbnail && mounted) {
        setImageUrl(asset.thumbnail);
      }

      // Then load full image
      const url = await getAssetUrl(settings.assetId);
      if (mounted && url) {
        setImageUrl(url);
      }
    };

    loadUrl();

    return () => {
      mounted = false;
    };
  }, [settings.assetId, getAssetUrl, getAssetById]);

  // Don't render if disabled or no asset
  if (!settings.enabled || !settings.assetId || !imageUrl) {
    return null;
  }

  const positionClass = POSITION_CLASSES[settings.position];
  const marginStyle =
    settings.position === 'center'
      ? {}
      : {
          margin: `${settings.margin}px`,
        };

  return (
    <div
      className={`absolute pointer-events-none z-10 ${positionClass}`}
      style={marginStyle}
    >
      <img
        src={imageUrl}
        alt="Watermark"
        className="max-w-none"
        style={{
          width: `${settings.size}%`,
          height: 'auto',
          opacity: settings.opacity / 100,
        }}
        draggable={false}
      />
    </div>
  );
}

export default WatermarkOverlay;
