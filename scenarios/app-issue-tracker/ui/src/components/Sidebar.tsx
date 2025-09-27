import { LucideIcon, GripVertical, Menu } from 'lucide-react';
import { NavKey } from '../data/sampleData';

interface SidebarItem {
  key: NavKey;
  label: string;
  icon: LucideIcon;
}

interface SidebarProps {
  collapsed: boolean;
  items: SidebarItem[];
  activeItem: NavKey;
  onSelect: (key: NavKey) => void;
  onToggle: () => void;
}

export function Sidebar({ collapsed, items, activeItem, onSelect, onToggle }: SidebarProps) {
  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
      <div className="sidebar-header">
        <button className="sidebar-toggle" onClick={onToggle} aria-label="Toggle navigation">
          <Menu size={18} />
        </button>
        {!collapsed && <span className="sidebar-title">Navigation</span>}
      </div>
      <nav className="sidebar-nav">
        {items.map((item) => {
          const Icon = item.icon;
          const active = item.key === activeItem;
          return (
            <button
              key={item.key}
              className={`sidebar-nav-item ${active ? 'active' : ''}`}
              onClick={() => onSelect(item.key)}
            >
              <Icon size={18} className="sidebar-icon" />
              {!collapsed && <span>{item.label}</span>}
            </button>
          );
        })}
      </nav>
      <div className="sidebar-footer">
        <GripVertical size={16} />
        {!collapsed && <span>App Issue Tracker</span>}
      </div>
    </aside>
  );
}
