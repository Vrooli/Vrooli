import { Settings, Cookie, Database, Fingerprint, Zap, ShieldCheck, Sparkles } from 'lucide-react';

export type SectionId = 'presets' | 'fingerprint' | 'behavior' | 'anti-detection' | 'cookies' | 'local-storage';

interface SectionGroup {
  label: string;
  sections: {
    id: SectionId;
    label: string;
    icon: React.ReactNode;
  }[];
}

const SECTION_GROUPS: SectionGroup[] = [
  {
    label: 'Settings',
    sections: [
      { id: 'presets', label: 'Presets', icon: <Sparkles size={16} /> },
      { id: 'fingerprint', label: 'Fingerprint', icon: <Fingerprint size={16} /> },
      { id: 'behavior', label: 'Behavior', icon: <Zap size={16} /> },
      { id: 'anti-detection', label: 'Anti-Detection', icon: <ShieldCheck size={16} /> },
    ],
  },
  {
    label: 'Storage',
    sections: [
      { id: 'cookies', label: 'Cookies', icon: <Cookie size={16} /> },
      { id: 'local-storage', label: 'LocalStorage', icon: <Database size={16} /> },
    ],
  },
];

interface SessionSidebarProps {
  activeSection: SectionId;
  onSectionChange: (section: SectionId) => void;
  hasStorageState?: boolean;
  cookieCount?: number;
  localStorageCount?: number;
}

export function SessionSidebar({
  activeSection,
  onSectionChange,
  hasStorageState,
  cookieCount,
  localStorageCount,
}: SessionSidebarProps) {
  return (
    <nav className="w-44 flex-shrink-0 border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 overflow-y-auto">
      {SECTION_GROUPS.map((group) => (
        <div key={group.label} className="py-3">
          <div className="px-4 mb-2 flex items-center gap-2">
            {group.label === 'Settings' ? (
              <Settings size={12} className="text-gray-400" />
            ) : (
              <Database size={12} className="text-gray-400" />
            )}
            <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
              {group.label}
            </span>
            {group.label === 'Storage' && hasStorageState && (
              <span className="ml-auto w-2 h-2 rounded-full bg-green-500" title="Storage data available" />
            )}
          </div>
          <ul className="space-y-0.5">
            {group.sections.map((section) => {
              const isActive = activeSection === section.id;
              const count =
                section.id === 'cookies' ? cookieCount : section.id === 'local-storage' ? localStorageCount : undefined;

              return (
                <li key={section.id}>
                  <button
                    onClick={() => onSectionChange(section.id)}
                    className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 transition-colors ${
                      isActive
                        ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 font-medium'
                        : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                    }`}
                  >
                    <span className={isActive ? 'text-blue-600 dark:text-blue-400' : ''}>{section.icon}</span>
                    <span className="flex-1">{section.label}</span>
                    {count !== undefined && count > 0 && (
                      <span
                        className={`text-[10px] px-1.5 py-0.5 rounded-full ${
                          isActive
                            ? 'bg-blue-200 dark:bg-blue-800 text-blue-700 dark:text-blue-300'
                            : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                        }`}
                      >
                        {count}
                      </span>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}

/** Check if a section is a settings section (editable) */
export function isSettingsSection(section: SectionId): boolean {
  return ['presets', 'fingerprint', 'behavior', 'anti-detection'].includes(section);
}
