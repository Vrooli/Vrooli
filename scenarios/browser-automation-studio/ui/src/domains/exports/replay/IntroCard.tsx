/**
 * IntroCard - Full-screen intro slide for replays
 */

import { useEffect, useState } from 'react';
import type { IntroCardSettings } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';

interface IntroCardProps {
  settings: IntroCardSettings;
}

export function IntroCard({ settings }: IntroCardProps) {
  const { getAssetUrl, getAssetById } = useAssetStore();
  const [logoUrl, setLogoUrl] = useState<string | null>(null);
  const [backgroundUrl, setBackgroundUrl] = useState<string | null>(null);

  // Load logo URL
  useEffect(() => {
    let mounted = true;

    const loadUrl = async () => {
      if (!settings.logoAssetId) {
        setLogoUrl(null);
        return;
      }

      // Try thumbnail first
      const asset = getAssetById(settings.logoAssetId);
      if (asset?.thumbnail && mounted) {
        setLogoUrl(asset.thumbnail);
      }

      // Load full image
      const url = await getAssetUrl(settings.logoAssetId);
      if (mounted && url) {
        setLogoUrl(url);
      }
    };

    loadUrl();
    return () => {
      mounted = false;
    };
  }, [settings.logoAssetId, getAssetUrl, getAssetById]);

  // Load background URL
  useEffect(() => {
    let mounted = true;

    const loadUrl = async () => {
      if (!settings.backgroundAssetId) {
        setBackgroundUrl(null);
        return;
      }

      // Try thumbnail first
      const asset = getAssetById(settings.backgroundAssetId);
      if (asset?.thumbnail && mounted) {
        setBackgroundUrl(asset.thumbnail);
      }

      // Load full image
      const url = await getAssetUrl(settings.backgroundAssetId);
      if (mounted && url) {
        setBackgroundUrl(url);
      }
    };

    loadUrl();
    return () => {
      mounted = false;
    };
  }, [settings.backgroundAssetId, getAssetUrl, getAssetById]);

  if (!settings.enabled) {
    return null;
  }

  const backgroundStyle: React.CSSProperties = backgroundUrl
    ? {
        backgroundImage: `url(${backgroundUrl})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
      }
    : {
        backgroundColor: settings.backgroundColor,
      };

  return (
    <div
      className="absolute inset-0 z-20 flex flex-col items-center justify-center p-8"
      style={backgroundStyle}
    >
      {/* Overlay for better text readability */}
      {backgroundUrl && (
        <div className="absolute inset-0 bg-black/40" />
      )}

      <div className="relative z-10 flex flex-col items-center text-center max-w-2xl">
        {/* Logo */}
        {logoUrl && (
          <img
            src={logoUrl}
            alt="Logo"
            className="max-w-[200px] max-h-[80px] mb-6 object-contain"
            draggable={false}
          />
        )}

        {/* Title */}
        {settings.title && (
          <h1
            className="text-4xl font-bold mb-4 leading-tight"
            style={{ color: settings.textColor }}
          >
            {settings.title}
          </h1>
        )}

        {/* Subtitle */}
        {settings.subtitle && (
          <p
            className="text-xl opacity-80"
            style={{ color: settings.textColor }}
          >
            {settings.subtitle}
          </p>
        )}
      </div>
    </div>
  );
}

export default IntroCard;
