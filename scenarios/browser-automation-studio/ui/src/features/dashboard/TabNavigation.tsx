import React from 'react';
import { Home, Activity, FolderOpen, Film, CalendarClock } from 'lucide-react';

export type DashboardTab = 'home' | 'executions' | 'exports' | 'projects' | 'schedules';

interface TabNavigationProps {
  activeTab: DashboardTab;
  onTabChange: (tab: DashboardTab) => void;
  runningCount?: number;
}

const tabs: Array<{ id: DashboardTab; label: string; icon: React.ReactNode }> = [
  { id: 'home', label: 'Home', icon: <Home size={16} /> },
  { id: 'projects', label: 'Projects', icon: <FolderOpen size={16} /> },
  { id: 'executions', label: 'Executions', icon: <Activity size={16} /> },
  { id: 'schedules', label: 'Schedules', icon: <CalendarClock size={16} /> },
  { id: 'exports', label: 'Exports', icon: <Film size={16} /> },
];

export const TabNavigation: React.FC<TabNavigationProps> = ({
  activeTab,
  onTabChange,
  runningCount = 0,
}) => {
  return (
    <nav className="flex items-center gap-1 border-b border-gray-800 px-4 sm:px-6 bg-flow-bg">
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id;
        const showBadge = tab.id === 'executions' && runningCount > 0;

        return (
          <button
            key={tab.id}
            data-testid={`dashboard-tab-${tab.id}`}
            onClick={() => onTabChange(tab.id)}
            className={`
              relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors
              border-b-2 -mb-[2px]
              ${
                isActive
                  ? 'text-surface border-flow-accent'
                  : 'text-gray-400 border-transparent hover:text-gray-200 hover:border-gray-600'
              }
            `}
            aria-selected={isActive}
            role="tab"
          >
            {tab.icon}
            <span>{tab.label}</span>
            {showBadge && (
              <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold text-white bg-green-500 rounded-full animate-pulse">
                {runningCount > 9 ? '9+' : runningCount}
              </span>
            )}
          </button>
        );
      })}
    </nav>
  );
};

export default TabNavigation;
