import { useEffect, useMemo, useState } from 'react';
import { useAssetStore } from '@/stores/assetStore';
import type { ReplayBackgroundSource } from './model';

export const useResolvedReplayBackground = (background: ReplayBackgroundSource): ReplayBackgroundSource => {
  const { getAssetUrl } = useAssetStore();
  const [resolvedUrl, setResolvedUrl] = useState<string | null>(null);
  const imageBackground = background.type === 'image' ? background : null;
  const imageAssetId = imageBackground?.assetId;
  const imageUrl = imageBackground?.url;

  useEffect(() => {
    let isCancelled = false;

    if (!imageBackground) {
      setResolvedUrl(null);
      return () => {
        isCancelled = true;
      };
    }

    if (imageUrl) {
      setResolvedUrl(imageUrl);
      return () => {
        isCancelled = true;
      };
    }

    if (!imageAssetId) {
      setResolvedUrl(null);
      return () => {
        isCancelled = true;
      };
    }

    void (async () => {
      const url = await getAssetUrl(imageAssetId);
      if (!isCancelled) {
        setResolvedUrl(url);
      }
    })();

    return () => {
      isCancelled = true;
    };
  }, [getAssetUrl, imageAssetId, imageBackground, imageUrl]);

  return useMemo(() => {
    if (!imageBackground) {
      return background;
    }
    if (imageUrl) {
      return imageBackground;
    }
    if (!imageAssetId || !resolvedUrl) {
      return imageBackground;
    }
    return { ...imageBackground, url: resolvedUrl };
  }, [background, imageAssetId, imageBackground, imageUrl, resolvedUrl]);
};
