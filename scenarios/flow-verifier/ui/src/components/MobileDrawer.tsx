import { type ReactNode, useEffect } from "react";
import { X } from "lucide-react";

import { useTranslation } from "../i18n";

interface Props {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
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
      data-testid="mobile-drawer-root"
      className="fixed inset-0 z-40 flex md:hidden"
      role="dialog"
      aria-modal="true"
      aria-label={t("nav.drawerLabel", { defaultValue: "Navigation drawer" })}
    >
      <button
        type="button"
        data-testid="mobile-drawer-backdrop"
        aria-label={t("nav.closeDrawer", { defaultValue: "Close navigation" })}
        className="absolute inset-0 cursor-default"
        style={{ background: "color-mix(in srgb, var(--color-shell) 60%, transparent)" }}
        onClick={onClose}
      />
      <aside
        data-testid="mobile-drawer-panel"
        className="pt-safe pb-safe relative z-10 flex h-full w-72 flex-col bg-app-surface shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-app-border px-3 py-3">
          <span className="text-sm font-semibold text-app-foreground">
            {t("app.brand", { defaultValue: "Flow Studio" })}
          </span>
          <button
            type="button"
            data-testid="mobile-drawer-close"
            aria-label={t("nav.closeDrawer", { defaultValue: "Close navigation" })}
            onClick={onClose}
            className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            <X aria-hidden className="h-5 w-5" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-auto px-2 py-3">{children}</div>
      </aside>
    </div>
  );
}
