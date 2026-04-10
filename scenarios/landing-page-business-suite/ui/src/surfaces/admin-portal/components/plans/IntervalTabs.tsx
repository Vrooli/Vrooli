import { cn } from '../../../../shared/lib/utils';

export type IntervalTab = 'month' | 'year' | 'other';

interface IntervalTabsProps {
  activeTab: IntervalTab;
  onTabChange: (tab: IntervalTab) => void;
  className?: string;
}

const TAB_LABELS: Record<IntervalTab, string> = {
  month: 'Monthly',
  year: 'Yearly',
  other: 'Other',
};

export function IntervalTabs({ activeTab, onTabChange, className }: IntervalTabsProps) {
  return (
    <div className={cn('flex overflow-hidden rounded-lg border border-white/10', className)}>
      {(['month', 'year', 'other'] as const).map((tab) => (
        <button
          key={tab}
          className={cn(
            'px-3 py-1 text-sm transition-colors',
            activeTab === tab
              ? 'bg-white/10 text-white'
              : 'bg-transparent text-slate-300 hover:bg-white/5'
          )}
          onClick={() => onTabChange(tab)}
        >
          {TAB_LABELS[tab]}
        </button>
      ))}
    </div>
  );
}
