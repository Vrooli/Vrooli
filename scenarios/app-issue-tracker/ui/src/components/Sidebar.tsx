import { LucideIcon } from 'lucide-react';
import { NavKey } from '../data/sampleData';

interface SidebarItem {
  key: NavKey;
  label: string;
  icon: LucideIcon;
}

interface SidebarProps {
  items: SidebarItem[];
  activeItem: NavKey;
  onSelect: (key: NavKey) => void;
}

export function Sidebar({ items, activeItem, onSelect }: SidebarProps) {
  return (
    <aside className="sidebar collapsed">
      <nav className="sidebar-nav">
        {items.map((item) => {
          const Icon = item.icon;
          const active = item.key === activeItem;
          return (
            <button
              key={item.key}
              className={`sidebar-nav-item ${active ? 'active' : ''}`}
              onClick={() => onSelect(item.key)}
              title={item.label}
            >
              <Icon size={18} className="sidebar-icon" />
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
