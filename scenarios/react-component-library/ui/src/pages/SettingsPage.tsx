/**
 * SettingsPage — preferences surface.
 *
 * Scoped to the preferences this scenario can persist today: theme
 * (light/dark/system) and UI locale. Theme is persisted via the
 * ThemeProvider's localStorage cache; locale via the i18n side-effect
 * `setLocale`. Server-backed preferences (density, font scale, etc.)
 * are a future addition — when added, layer in a `lib/preferences`
 * module and a server-side store.
 */
import { type Theme } from "../components/theme/theme-context";
import { useTheme } from "../components/theme/useTheme";
import { Select } from "../components/ui/select";
import {
  SUPPORTED_LOCALES,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
  useTranslation,
} from "../i18n";

export function SettingsPage() {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();
  const locale = getCurrentLocale();

  return <div data-testid="settings-page" className="grid max-w-5xl gap-6 md:grid-cols-[12rem_minmax(0,1fr)]">
    <nav aria-label={t("settings.preferences", { defaultValue: "Preferences" })} className="rounded-panel border border-app-border bg-app-surface p-2">
      <p className="px-2 py-2 text-sm font-semibold">{t("settings.preferences", { defaultValue: "Preferences" })}</p>
      <a href="#appearance" className="block rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted">{t("settings.appearance", { defaultValue: "Appearance" })}</a>
      <a href="#language" className="block rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted">{t("settings.language", { defaultValue: "Language" })}</a>
      <span className="block cursor-not-allowed rounded-control px-2 py-2 text-sm text-app-muted-foreground" aria-disabled="true">{t("settings.workspaceFuture", { defaultValue: "Workspace (future)" })}</span>
    </nav>
    <div className="flex flex-col gap-6">
      <Section id="appearance" title={t("settings.appearance", { defaultValue: "Appearance" })}>
        <Field label={t("settings.theme", { defaultValue: "Theme" })}>
          <AppearanceCards
            testid="settings-theme"
            value={theme}
            options={[
              { value: "light", label: t("theme.light", { defaultValue: "Light" }) },
              { value: "dark", label: t("theme.dark", { defaultValue: "Dark" }) },
              { value: "system", label: t("theme.system", { defaultValue: "System" }) },
            ]}
            onChange={(v) => setTheme(v as Theme)}
          />
        </Field>
        <p className="text-xs text-app-muted-foreground">{t("settings.densityFuture", { defaultValue: "Density will appear here when it is a real preference." })}</p>
      </Section>

      <Section id="language" title={t("settings.language", { defaultValue: "Language" })}>
        <Field label={t("locale.switcherLabel", { defaultValue: "Language" })}>
          <Select
            data-testid="settings-locale"
            value={locale}
            options={SUPPORTED_LOCALES.map((code) => ({ value: code, label: getLocaleConfig(code).nativeLabel }))}
            onChange={(event) => void setLocale(event.target.value as (typeof SUPPORTED_LOCALES)[number])}
            className="w-auto min-w-40"
          />
        </Field>
      </Section>
      <p className="text-sm text-app-muted-foreground">{t("settings.savedBrowser", { defaultValue: "Saved automatically in this browser." })}</p>
    </div>
  </div>;
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="rounded-panel border border-app-border bg-app-surface p-4">
      <h2 className="text-sm font-semibold text-app-foreground">{title}</h2>
      <div className="mt-3 flex flex-col gap-3">{children}</div>
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-wrap items-center justify-between gap-2 text-sm text-app-foreground">
      <span className="text-app-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

function AppearanceCards({
  testid,
  value,
  options,
  onChange,
}: {
  testid: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (v: string) => void;
}) {
  return (
    <div
      data-testid={testid}
      role="radiogroup"
      className="grid w-full gap-2 sm:grid-cols-3"
    >
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={active}
            data-testid={`${testid}-${o.value}`}
            onClick={() => onChange(o.value)}
            className={[
              "min-h-24 rounded-panel border p-3 text-left text-sm",
              active
                ? "border-app-primary bg-app-surface-muted text-app-foreground ring-1 ring-app-primary"
                : "border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
            ].join(" ")}
          >
            <span className="block font-medium">{o.label}</span>
          </button>
        );
      })}
    </div>
  );
}
