/**
 * MobileDrawer — slide-in panel hosting the full nav surface for mobile
 * viewports. Triggered from `MobileHeader` via the menu button.
 */
import { type ReactNode, useEffect } from "react";
import { X } from "lucide-react";
import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

interface Props {
  open: boolean;
  onClose: () => void;
  children?: ReactNode;
}

export function MobileDrawer({ open, onClose, children }: Props) {
  const { t } = useTranslation();

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      data-testid={selectors.layout.drawer}
      className="fixed inset-0 z-40 flex md:hidden"
      role="dialog"
      aria-modal="true"
      aria-label={t(strings.layout.drawerLabel)}
    >
      <button
        type="button"
        data-testid={selectors.layout.drawerBackdrop}
        aria-label={t(strings.layout.closeDrawer)}
        className="absolute inset-0 cursor-default"
        style={{ background: "color-mix(in srgb, var(--color-shell) 60%, transparent)" }}
        onClick={onClose}
      />
      <aside
        data-testid="layout-drawer-panel"
        className="pt-safe pb-safe relative z-10 flex h-full w-72 flex-col bg-app-surface shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-app-border px-3 py-3">
          <span className="text-sm font-semibold text-app-foreground">
            {t(strings.app.brand)}
          </span>
          <button
            type="button"
            data-testid={selectors.layout.drawerClose}
            aria-label={t(strings.layout.closeDrawer)}
            onClick={onClose}
            className="inline-flex h-touch w-touch items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            <X aria-hidden className="h-5 w-5" />
          </button>
        </div>
        <nav
          className="flex min-h-0 flex-1 flex-col gap-1 overflow-auto px-2 py-3"
          aria-label={t(strings.layout.drawerLabel)}
        >
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.key}
                to={item.path}
                end={item.end}
                onClick={onClose}
                data-testid={selectors.layout.drawerLink({ key: item.key })}
                className={({ isActive }) =>
                  [
                    "flex items-center gap-2 rounded-control px-3 py-2 text-sm",
                    isActive
                      ? "bg-app-surface-muted font-medium text-app-foreground"
                      : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                  ].join(" ")
                }
              >
                <Icon aria-hidden className="h-4 w-4" />
                <span>{t(item.labelKey)}</span>
              </NavLink>
            );
          })}
        </nav>
        {children}
      </aside>
    </div>
  );
}
