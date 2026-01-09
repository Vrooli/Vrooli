import { useState } from 'react';
import { ChevronDown, ChevronRight, HelpCircle } from 'lucide-react';
import Tooltip from '@shared/ui/Tooltip';

interface SettingSectionProps {
  title: string;
  tooltip?: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

export function SettingSection({ title, tooltip, defaultOpen = true, children }: SettingSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          <span className="font-medium text-surface">{title}</span>
          {tooltip && (
            <Tooltip content={tooltip}>
              <HelpCircle size={14} className="text-gray-500" />
            </Tooltip>
          )}
        </div>
        {isOpen ? <ChevronDown size={16} className="text-gray-400" /> : <ChevronRight size={16} className="text-gray-400" />}
      </button>
      {isOpen && <div className="p-4 bg-gray-900/50 space-y-4">{children}</div>}
    </div>
  );
}
