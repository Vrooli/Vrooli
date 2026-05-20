/**
 * InspectorPanel — right-pane detail surface on desktop, bottom sheet on
 * mobile. Used by detail pages (validation, surface, reindex job) to
 * display the "currently selected" record without pushing the main
 * content out of view.
 */
import { type ReactNode, useEffect } from "react";
import { X } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useIsMobile } from "../hooks/useMediaQuery";
import { useTranslation } from "../i18n";

interface Props {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function InspectorPanel({ open, onClose, title, children }: Props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  if (isMobile) {
    return (
      <div
        data-testid={selectors.inspector.panel}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="fixed inset-x-0 bottom-0 z-40 flex max-h-[80vh] flex-col md:hidden"
      >
        <button
          type="button"
          data-testid={selectors.inspector.backdrop}
          aria-label={t(strings.inspector.close)}
          onClick={onClose}
          className="absolute inset-0 cursor-default"
          style={{ background: "color-mix(in srgb, var(--color-shell) 60%, transparent)" }}
        />
        <section className="pb-safe relative z-10 mt-auto flex max-h-[80vh] flex-col rounded-t-panel border-t border-app-border bg-app-surface shadow-2xl">
          <Header title={title} onClose={onClose} />
          <div className="min-h-0 flex-1 overflow-auto px-3 py-3">{children}</div>
        </section>
      </div>
    );
  }

  return (
    <aside
      data-testid={selectors.inspector.panel}
      aria-label={title}
      className="hidden h-full w-80 shrink-0 flex-col border-l border-app-border bg-app-surface md:flex"
    >
      <Header title={title} onClose={onClose} />
      <div className="min-h-0 flex-1 overflow-auto px-3 py-3">{children}</div>
    </aside>
  );
}

function Header({ title, onClose }: { title: string; onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-between border-b border-app-border px-3 py-2">
      <h3 data-testid={selectors.inspector.title} className="text-sm font-semibold text-app-foreground">
        {title}
      </h3>
      <button
        type="button"
        data-testid={selectors.inspector.close}
        aria-label={t(strings.inspector.close)}
        onClick={onClose}
        className="inline-flex h-touch w-touch items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
      >
        <X aria-hidden className="h-4 w-4" />
      </button>
    </div>
  );
}
