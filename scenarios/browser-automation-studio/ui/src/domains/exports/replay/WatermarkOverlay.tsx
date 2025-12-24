/**
 * WatermarkOverlay - Renders watermark on replay player
 *
 * Supports both user-uploaded assets and built-in assets (like Vrooli Ascension)
 */

import { useEffect, useMemo, useState } from 'react';
import type { WatermarkSettings, WatermarkPosition } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import { useOverlayRegistry } from '@/domains/replay-positioning';
import { isBuiltInAssetId, getBuiltInAsset } from '@lib/builtInAssets';

interface WatermarkOverlayProps {
  settings: WatermarkSettings;
}

const DEFAULT_CONTAINER_ID = 'browser-content';

export function WatermarkOverlay({ settings }: WatermarkOverlayProps) {
  const { getAssetUrl, getAssetById } = useAssetStore();
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [naturalSize, setNaturalSize] = useState<{ width: number; height: number } | null>(null);
  const registry = useOverlayRegistry();

  // Load image URL when asset ID changes
  useEffect(() => {
    let mounted = true;

    const loadUrl = async () => {
      if (!settings.assetId) {
        setImageUrl(null);
        return;
      }

      // Check if it's a built-in asset first
      if (isBuiltInAssetId(settings.assetId)) {
        const builtInAsset = getBuiltInAsset(settings.assetId);
        if (builtInAsset && mounted) {
          setImageUrl(builtInAsset.url);
          // Set natural size from built-in asset metadata
          setNaturalSize({ width: builtInAsset.width, height: builtInAsset.height });
        }
        return;
      }

      // For user assets, first try to use thumbnail for quick display
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
  const containerRect = registry?.getRect(DEFAULT_CONTAINER_ID);
  const resolvedSize = useMemo(() => {
    if (!containerRect || !naturalSize) {
      return null;
    }
    const width = (containerRect.width * settings.size) / 100;
    const ratio = naturalSize.width > 0 ? naturalSize.height / naturalSize.width : 1;
    return {
      width,
      height: width * ratio,
    };
  }, [containerRect, naturalSize, settings.size]);

  const anchorRect = useMemo(() => {
    if (!registry || !resolvedSize) {
      return null;
    }
    const rect = registry.resolveAnchor({
      id: 'watermark',
      containerId: DEFAULT_CONTAINER_ID,
      position: settings.position as WatermarkPosition,
      size: resolvedSize,
      margin: settings.margin,
    });
    if (rect) {
      registry.setRect('watermark', rect);
    }
    return rect;
  }, [registry, resolvedSize, settings.margin, settings.position]);

  if (!settings.enabled || !settings.assetId || !imageUrl) {
    return null;
  }

  if (!containerRect || !resolvedSize || !anchorRect) {
    return (
      <img
        src={imageUrl}
        alt="Watermark"
        style={{ display: 'none' }}
        onLoad={(event) => {
          const target = event.currentTarget;
          if (target.naturalWidth && target.naturalHeight) {
            setNaturalSize({ width: target.naturalWidth, height: target.naturalHeight });
          }
        }}
        draggable={false}
      />
    );
  }

  return (
    <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none', zIndex: 20 }}>
      <div
        style={{
          position: 'absolute',
          left: `${anchorRect.x}px`,
          top: `${anchorRect.y}px`,
          width: `${resolvedSize.width}px`,
          height: `${resolvedSize.height}px`,
        }}
      >
        <img
          src={imageUrl}
          alt="Watermark"
          style={{
            display: 'block',
            width: '100%',
            height: '100%',
            opacity: settings.opacity / 100,
            maxWidth: 'none',
          }}
          onLoad={(event) => {
            const target = event.currentTarget;
            if (target.naturalWidth && target.naturalHeight) {
              setNaturalSize({ width: target.naturalWidth, height: target.naturalHeight });
            }
          }}
          draggable={false}
        />
      </div>
    </div>
  );
}

export default WatermarkOverlay;
