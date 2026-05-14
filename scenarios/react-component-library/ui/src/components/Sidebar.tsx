import { type ReactNode } from "react";
import { NavLink, Link } from "react-router-dom";
import { GaugeCircle, GitBranch, Library, Settings as SettingsIcon } from "lucide-react";

import { useTranslation } from "../i18n";

interface NavItem {
  to: string;
  label: string;
  end?: boolean;
  icon: ReactNode;
  testid: string;
}

interface SidebarContentProps {
  onNavigate?: () => void;
  headerSlot?: ReactNode;
  inventorySlot?: ReactNode;
}

export function SidebarContent({ onNavigate, headerSlot, inventorySlot }: SidebarContentProps) {
  const { t } = useTranslation();

  const items: NavItem[] = [
    {
      to: "/",
      end: true,
      label: t("nav.dashboard", { defaultValue: "Dashboard" }),
      icon: <GaugeCircle aria-hidden className="h-4 w-4" />,
      testid: "nav-dashboard",
    },
    {
      to: "/components",
      label: t("nav.components", { defaultValue: "Components" }),
      icon: <Library aria-hidden className="h-4 w-4" />,
      testid: "nav-components",
    },
    {
      to: "/adoptions",
      label: t("nav.adoptions", { defaultValue: "Adoptions" }),
      icon: <GitBranch aria-hidden className="h-4 w-4" />,
      testid: "nav-adoptions",
    },
    {
      to: "/settings",
      label: t("nav.settings", { defaultValue: "Settings" }),
      icon: <SettingsIcon aria-hidden className="h-4 w-4" />,
      testid: "nav-settings",
    },
  ];

  return (
    <div data-testid="app-sidebar-content" className="flex min-h-0 flex-1 flex-col">
      <div className="hidden items-center gap-2 border-b border-app-border px-4 py-4 md:flex">
        <Link
          to="/"
          onClick={onNavigate}
          className="flex items-center gap-2 text-app-foreground"
          data-testid="app-brand"
        >
          <span
            aria-hidden
            className="inline-flex h-7 w-7 items-center justify-center rounded-control bg-app-primary text-sm font-semibold text-app-primary-foreground"
          >
            {t("app.brandInitials", { defaultValue: "RC" })}
          </span>
          <span className="text-sm font-semibold tracking-tight">
            {t("app.brand", { defaultValue: "Component Library" })}
          </span>
        </Link>
        <div className="ms-auto">{headerSlot}</div>
      </div>

      <nav className="flex flex-col gap-1 px-2 py-3" aria-label={t("nav.label", { defaultValue: "Primary navigation" })}>
        {items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            onClick={onNavigate}
            data-testid={item.testid}
            className={({ isActive }) =>
              [
                "flex items-center gap-2 rounded-control px-3 py-2 text-sm",
                isActive
                  ? "bg-app-surface-muted font-medium text-app-foreground"
                  : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
              ].join(" ")
            }
          >
            {item.icon}
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="min-h-0 flex-1 overflow-auto border-t border-app-border px-2 py-3">
        {inventorySlot}
      </div>
    </div>
  );
}
