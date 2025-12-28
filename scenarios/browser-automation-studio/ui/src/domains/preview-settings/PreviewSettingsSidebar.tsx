import { Zap, Eye, MousePointer2, Play, Sparkles } from 'lucide-react';
import type { SectionId, SectionGroup } from './types';

/** Section groups with their configurations */
const SECTION_GROUPS: SectionGroup[] = [
  {
    id: 'stream',
    label: 'Stream',
    sections: [
      { id: 'stream', label: 'Quality', icon: <Zap size={16} /> },
    ],
  },
  {
    id: 'replay',
    label: 'Replay Style',
    sections: [
      { id: 'visual', label: 'Visual', icon: <Eye size={16} /> },
      { id: 'cursor', label: 'Cursor', icon: <MousePointer2 size={16} /> },
      { id: 'playback', label: 'Playback', icon: <Play size={16} /> },
      { id: 'branding', label: 'Branding', icon: <Sparkles size={16} /> },
    ],
  },
];

interface PreviewSettingsSidebarProps {
  activeSection: SectionId;
  onSectionChange: (section: SectionId) => void;
}

export function PreviewSettingsSidebar({
  activeSection,
  onSectionChange,
}: PreviewSettingsSidebarProps) {
  return (
    <nav className="w-44 flex-shrink-0 border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 overflow-y-auto">
      {SECTION_GROUPS.map((group) => (
        <div key={group.id} className="py-3">
          <div className="px-4 mb-2">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
              {group.label}
            </span>
          </div>
          <ul className="space-y-0.5">
            {group.sections.map((section) => {
              const isActive = activeSection === section.id;

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
                    <span className={isActive ? 'text-blue-600 dark:text-blue-400' : ''}>
                      {section.icon}
                    </span>
                    <span className="flex-1">{section.label}</span>
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

/** Get all section IDs for validation */
export function getAllSectionIds(): SectionId[] {
  return SECTION_GROUPS.flatMap((group) => group.sections.map((s) => s.id));
}
