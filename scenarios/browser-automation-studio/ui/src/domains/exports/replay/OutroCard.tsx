/**
 * OutroCard - Full-screen outro slide for replays with CTA
 */

import { useEffect, useState } from 'react';
import { ExternalLink } from 'lucide-react';
import type { OutroCardSettings } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';

interface OutroCardProps {
  settings: OutroCardSettings;
}

export function OutroCard({ settings }: OutroCardProps) {
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

      const asset = getAssetById(settings.logoAssetId);
      if (asset?.thumbnail && mounted) {
        setLogoUrl(asset.thumbnail);
      }

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

      const asset = getAssetById(settings.backgroundAssetId);
      if (asset?.thumbnail && mounted) {
        setBackgroundUrl(asset.thumbnail);
      }

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
            className="text-4xl font-bold mb-6 leading-tight"
            style={{ color: settings.textColor }}
          >
            {settings.title}
          </h1>
        )}

        {/* CTA Button */}
        {settings.ctaText && settings.ctaUrl && (
          <a
            href={settings.ctaUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl font-semibold text-lg transition-all hover:scale-105 hover:shadow-lg"
            style={{
              backgroundColor: settings.textColor,
              color: settings.backgroundColor,
            }}
          >
            {settings.ctaText}
            <ExternalLink size={18} />
          </a>
        )}

        {/* CTA text only (no URL) */}
        {settings.ctaText && !settings.ctaUrl && (
          <p
            className="text-xl opacity-80"
            style={{ color: settings.textColor }}
          >
            {settings.ctaText}
          </p>
        )}
      </div>
    </div>
  );
}

export default OutroCard;
