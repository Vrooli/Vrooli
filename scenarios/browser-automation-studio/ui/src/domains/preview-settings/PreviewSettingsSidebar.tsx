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

/** Flat list of all sections for the horizontal tabs */
const ALL_SECTIONS = SECTION_GROUPS.flatMap((group) => group.sections);

interface PreviewSettingsSidebarProps {
  activeSection: SectionId;
  onSectionChange: (section: SectionId) => void;
}

export function PreviewSettingsSidebar({
  activeSection,
  onSectionChange,
}: PreviewSettingsSidebarProps) {
  return (
    <nav
      className="flex items-center border-b border-gray-200 dark:border-gray-700 overflow-x-auto scrollbar-thin scrollbar-thumb-gray-300 dark:scrollbar-thumb-gray-600"
      role="tablist"
    >
      {ALL_SECTIONS.map((section) => {
        const isActive = activeSection === section.id;

        return (
          <button
            key={section.id}
            onClick={() => onSectionChange(section.id)}
            className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap transition-colors shrink-0 ${
              isActive
                ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-b-2 border-blue-500'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-800'
            }`}
            role="tab"
            aria-selected={isActive}
            aria-controls={`${section.id}-panel`}
          >
            <span className={isActive ? 'text-blue-600 dark:text-blue-400' : ''}>
              {section.icon}
            </span>
            <span>{section.label}</span>
          </button>
        );
      })}
    </nav>
  );
}
