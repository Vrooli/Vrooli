import React from 'react';
import {
  Cookie,
  Database,
  Fingerprint,
  Zap,
  ShieldCheck,
  Sparkles,
  Globe,
  FileText,
  Cog,
  History,
  Layers,
} from 'lucide-react';

export type SectionId =
  | 'presets'
  | 'fingerprint'
  | 'behavior'
  | 'anti-detection'
  | 'proxy'
  | 'extra-headers'
  | 'cookies'
  | 'local-storage'
  | 'service-workers'
  | 'history'
  | 'tabs';

interface SectionGroup {
  label: string;
  sections: {
    id: SectionId;
    label: string;
    icon: React.ReactNode;
  }[];
}

export const SECTION_GROUPS: SectionGroup[] = [
  {
    label: 'Settings',
    sections: [
      { id: 'presets', label: 'Presets', icon: <Sparkles size={16} /> },
      { id: 'fingerprint', label: 'Fingerprint', icon: <Fingerprint size={16} /> },
      { id: 'behavior', label: 'Behavior', icon: <Zap size={16} /> },
      { id: 'anti-detection', label: 'Anti-Detection', icon: <ShieldCheck size={16} /> },
      { id: 'proxy', label: 'Proxy', icon: <Globe size={16} /> },
      { id: 'extra-headers', label: 'HTTP Headers', icon: <FileText size={16} /> },
    ],
  },
  {
    label: 'Storage',
    sections: [
      { id: 'cookies', label: 'Cookies', icon: <Cookie size={16} /> },
      { id: 'local-storage', label: 'LocalStorage', icon: <Database size={16} /> },
      { id: 'service-workers', label: 'Service Workers', icon: <Cog size={16} /> },
      { id: 'history', label: 'History', icon: <History size={16} /> },
      { id: 'tabs', label: 'Tabs', icon: <Layers size={16} /> },
    ],
  },
];

const SETTINGS_SECTION_IDS = new Set<SectionId>(
  SECTION_GROUPS.find((group) => group.label === 'Settings')?.sections.map((section) => section.id) ?? []
);

export function isSettingsSection(section: SectionId): boolean {
  return SETTINGS_SECTION_IDS.has(section);
}
