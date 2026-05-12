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

  return (
    <div data-testid="settings-page" className="flex max-w-3xl flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("settings.title", { defaultValue: "Settings" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("settings.subtitle", {
            defaultValue:
              "Theme and locale preferences persist locally in your browser.",
          })}
        </p>
      </header>

      <Section title={t("settings.appearance", { defaultValue: "Appearance" })}>
        <Field label={t("settings.theme", { defaultValue: "Theme" })}>
          <Segmented
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
      </Section>

      <Section title={t("settings.language", { defaultValue: "Language" })}>
        <Field label={t("locale.switcherLabel", { defaultValue: "Language" })}>
          <Segmented
            testid="settings-locale"
            value={locale}
            options={SUPPORTED_LOCALES.map((code) => ({
              value: code,
              label: getLocaleConfig(code).nativeLabel,
            }))}
            onChange={(v) => void setLocale(v as (typeof SUPPORTED_LOCALES)[number])}
          />
        </Field>
      </Section>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4">
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

function Segmented({
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
      className="inline-flex overflow-hidden rounded-control border border-app-border"
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
              "h-9 px-3 text-xs",
              active
                ? "bg-app-primary text-app-primary-foreground"
                : "bg-app-surface text-app-foreground hover:bg-app-surface-muted",
            ].join(" ")}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
