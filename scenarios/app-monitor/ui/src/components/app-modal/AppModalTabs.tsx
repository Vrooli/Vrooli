import clsx from 'clsx';
import { Layers, Activity, AlertCircle, FileText, Zap, Award } from 'lucide-react';
import type { CompleteDiagnostics } from '@/types';

export type TabType = 'overview' | 'diagnostics' | 'tech-stack' | 'docs' | 'lighthouse' | 'completeness';

interface AppModalTabsProps {
  activeTab: TabType;
  onTabChange: (tab: TabType) => void;
  diagnostics: CompleteDiagnostics | null;
}

interface TabDef {
  id: TabType;
  label: string;
  icon: React.ReactNode;
  badge?: { count: number; variant: 'warn' | 'info' } | null;
}

/** Horizontal tab bar with icons and optional count badges. */
export default function AppModalTabs({ activeTab, onTabChange, diagnostics }: AppModalTabsProps) {
  const warningCount = diagnostics && Array.isArray(diagnostics.warnings) ? diagnostics.warnings.length : 0;
  const docsCount = diagnostics?.documents?.total ?? 0;

  const tabs: TabDef[] = [
    { id: 'overview', label: 'Overview', icon: <Layers size={16} aria-hidden /> },
    {
      id: 'diagnostics',
      label: 'Diagnostics',
      icon: <Activity size={16} aria-hidden />,
      badge: warningCount > 0 ? { count: warningCount, variant: 'warn' } : null,
    },
    { id: 'tech-stack', label: 'Tech Stack', icon: <AlertCircle size={16} aria-hidden /> },
    {
      id: 'docs',
      label: 'Docs',
      icon: <FileText size={16} aria-hidden />,
      badge: docsCount > 0 ? { count: docsCount, variant: 'info' } : null,
    },
    { id: 'lighthouse', label: 'Performance', icon: <Zap size={16} aria-hidden /> },
    { id: 'completeness', label: 'Completeness', icon: <Award size={16} aria-hidden /> },
  ];

  return (
    <div className="modal-tabs" role="tablist" aria-label="Application details">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`tabpanel-${tab.id}`}
          id={`tab-${tab.id}`}
          className={clsx('modal-tab', { 'modal-tab--active': activeTab === tab.id })}
          onClick={() => onTabChange(tab.id)}
        >
          {tab.icon}
          <span>{tab.label}</span>
          {tab.badge && (
            <span className={`modal-tab-badge modal-tab-badge--${tab.badge.variant}`}>
              {tab.badge.count}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
