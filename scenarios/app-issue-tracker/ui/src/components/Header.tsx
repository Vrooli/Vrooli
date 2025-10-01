import type { ReactNode } from 'react';
import { Bug, Menu } from 'lucide-react';

interface HeaderProps {
  sidebarCollapsed: boolean;
  onSidebarToggle: () => void;
  rightContent?: ReactNode;
}

export function Header({
  sidebarCollapsed,
  onSidebarToggle,
  rightContent,
}: HeaderProps) {
  return (
    <header className="app-header">
      <div className="header-left">
        <button
          type="button"
          className="header-sidebar-toggle"
          onClick={onSidebarToggle}
          aria-expanded={!sidebarCollapsed}
          aria-label={`${sidebarCollapsed ? 'Open' : 'Collapse'} navigation sidebar`}
        >
          <Menu size={20} />
        </button>
        <div className="scenario-badge">
          <Bug size={20} />
        </div>
        <div>
          <h1 className="scenario-title">App Issue Tracker</h1>
          <p className="scenario-subtitle">AI-assisted triage for Vrooli scenarios</p>
        </div>
      </div>
      {rightContent && <div className="header-right">{rightContent}</div>}
    </header>
  );
}
