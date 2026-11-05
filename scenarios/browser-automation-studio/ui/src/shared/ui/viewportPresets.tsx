import { Monitor, Smartphone, Tv } from 'lucide-react';
import type { ViewportPresetOption } from './viewportPresetTypes';

export const VIEWPORT_PRESETS: ViewportPresetOption[] = [
  {
    id: 'widescreen-720',
    label: '1280 × 720',
    description: 'YouTube, demos',
    width: 1280,
    height: 720,
    icon: <Monitor size={14} />,
    ratioLabel: '16:9',
  },
  {
    id: 'full-hd',
    label: '1920 × 1080',
    description: 'Desktop shares',
    width: 1920,
    height: 1080,
    icon: <Tv size={14} />,
    ratioLabel: '16:9',
  },
  {
    id: 'instagram-feed',
    label: '1080 × 1350',
    description: 'Instagram feed',
    width: 1080,
    height: 1350,
    icon: <Smartphone size={14} />,
    ratioLabel: '4:5',
  },
  {
    id: 'tiktok',
    label: '1080 × 1920',
    description: 'TikTok, Reels',
    width: 1080,
    height: 1920,
    icon: <Smartphone size={14} />,
    ratioLabel: '9:16',
  },
];
