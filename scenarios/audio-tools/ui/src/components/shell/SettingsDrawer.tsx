import * as React from "react";
import { X } from "lucide-react";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import {
  SUPPORTED_LOCALES,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
  useTranslation,
} from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { usePreferences } from "../../hooks/usePreferences";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function SettingsDrawer({ open, onClose }: Props) {
  const { t } = useTranslation();
  const { preferences, setTheme, setFontScale, setReducedMotion } = usePreferences();
  const currentLocale = getCurrentLocale();

  React.useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-drawer">
      <div
        className="absolute inset-0 bg-app-shell/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />
      <aside
        role="dialog"
        aria-modal="true"
        aria-label={t(strings.shell.settingsTitle)}
        className="absolute inset-y-0 right-0 flex w-full max-w-md flex-col border-l border-app-border bg-app-surface shadow-2xl md:w-96"
      >
        <header className="flex items-center justify-between border-b border-app-border px-4 py-3">
          <h2 className="text-sm font-semibold text-app-foreground">{t(strings.shell.settingsTitle)}</h2>
          <Button variant="ghost" size="icon" aria-label={t(strings.shell.closeSettings)} onClick={onClose}>
            <X className="h-5 w-5" aria-hidden="true" />
          </Button>
        </header>

        <div className="flex-1 overflow-y-auto p-4">
          <Field label={t(strings.settings.themeLabel)}>
            <Select
              value={preferences.theme}
              onChange={(e) => setTheme(e.currentTarget.value as "light" | "dark" | "system")}
              aria-label={t(strings.settings.themeLabel)}
            >
              <option value="light">{t(strings.settings.themeLight)}</option>
              <option value="dark">{t(strings.settings.themeDark)}</option>
              <option value="system">{t(strings.settings.themeSystem)}</option>
            </Select>
          </Field>

          <Field label={t(strings.settings.fontScaleLabel)}>
            <Select
              value={preferences.fontScale}
              onChange={(e) =>
                setFontScale(e.currentTarget.value as "compact" | "comfortable" | "large")
              }
              aria-label={t(strings.settings.fontScaleLabel)}
            >
              <option value="compact">{t(strings.settings.fontScaleCompact)}</option>
              <option value="comfortable">{t(strings.settings.fontScaleComfortable)}</option>
              <option value="large">{t(strings.settings.fontScaleLarge)}</option>
            </Select>
          </Field>

          <Field label={t(strings.settings.motionLabel)}>
            <label className="inline-flex items-center gap-2 text-sm text-app-foreground">
              <input
                type="checkbox"
                checked={preferences.reducedMotion}
                onChange={(e) => setReducedMotion(e.currentTarget.checked)}
                className="h-4 w-4 accent-app-primary"
              />
              {t(strings.settings.reducedMotion)}
            </label>
          </Field>

          <Field label={t(strings.locale.switcherLabel)}>
            <div
              role="group"
              aria-label={t(strings.locale.switcherLabel)}
              data-testid={selectors.locale.switcher}
              className="flex flex-wrap items-center gap-1 rounded-control border border-app-border bg-app-surface-muted p-1 text-xs"
            >
              {SUPPORTED_LOCALES.map((lng) => {
                const active = currentLocale === lng;
                return (
                  <button
                    key={lng}
                    type="button"
                    data-testid={selectors.locale.toggle({ code: lng })}
                    onClick={() => void setLocale(lng)}
                    aria-pressed={active}
                    className={
                      active
                        ? "rounded-control bg-app-primary px-3 py-1 font-medium text-app-primary-foreground"
                        : "rounded-control px-3 py-1 text-app-muted-foreground hover:text-app-foreground"
                    }
                  >
                    {getLocaleConfig(lng).nativeLabel}
                  </button>
                );
              })}
            </div>
          </Field>
        </div>
      </aside>
    </div>
  );
}

function Field({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
        {label}
      </p>
      {children}
    </div>
  );
}
