import { Link } from "react-router-dom";
import { cn } from "../../lib/utils";

interface TabNavItem {
  id: string;
  label: string;
  badgeCount?: number;
}

interface TabNavProps {
  tabs: TabNavItem[];
  activeTab: string;
  onChange: (id: string) => void;
  /** Base path for generating links. If not provided, uses button behavior without links. */
  basePath?: string;
}

export const TabNav = ({ tabs, activeTab, onChange, basePath }: TabNavProps) => {
  return (
    <nav className="flex flex-wrap gap-2">
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab;
        const className = cn(
          "flex items-center gap-2 rounded-full border px-4 py-2 text-sm transition",
          isActive
            ? "border-emerald-400 bg-emerald-500/20 text-white shadow-[0_10px_40px_-15px_rgba(16,185,129,0.4)]"
            : "border-white/20 text-white/70 hover:border-white/40 hover:text-white"
        );

        const content = (
          <>
            <span>{tab.label}</span>
            {typeof tab.badgeCount === "number" && tab.badgeCount > 0 ? (
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500/80 text-[11px] font-semibold text-slate-950">
                {tab.badgeCount}
              </span>
            ) : null}
          </>
        );

        // Use Link if basePath is provided, otherwise use button
        if (basePath) {
          const to = tab.id === "dashboard" ? "/" : `/${basePath}/${tab.id}`.replace(/\/+/g, "/");
          return (
            <Link
              key={tab.id}
              to={to}
              onClick={() => onChange(tab.id)}
              className={className}
            >
              {content}
            </Link>
          );
        }

        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            className={className}
          >
            {content}
          </button>
        );
      })}
    </nav>
  );
};
