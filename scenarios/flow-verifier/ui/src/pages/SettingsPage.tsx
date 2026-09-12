import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { type Theme } from "../components/theme/theme-context";
import { useTheme } from "../components/theme/useTheme";
import { useTranslation } from "../i18n";
import {
  DEFAULT_SETTINGS,
  fetchSettings,
  putSettings,
  readCache,
  type Density,
  type FontScale,
  type UserSettings,
} from "../lib/preferences";

const KEY = ["settings"];

export function SettingsPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const { setTheme } = useTheme();

  const { data, isLoading, error } = useQuery({
    queryKey: KEY,
    queryFn: fetchSettings,
    initialData: readCache(),
  });

  const settings: UserSettings = data ?? DEFAULT_SETTINGS;

  const mutation = useMutation({
    mutationFn: putSettings,
    onSuccess: (next) => qc.setQueryData(KEY, next),
  });

  const update = useCallback(
    (patch: Partial<UserSettings>) => {
      qc.setQueryData(KEY, { ...settings, ...patch });
      mutation.mutate(patch);
    },
    [mutation, qc, settings],
  );

  return (
    <div data-testid="settings-page" className="flex max-w-3xl flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("settings.title", { defaultValue: "Settings" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("settings.subtitle", {
            defaultValue: "Preferences are stored on the server and persist across sessions.",
          })}
        </p>
      </header>

      {isLoading && (
        <p data-testid="settings-loading" className="text-sm text-app-muted-foreground">
          {t("settings.loading", { defaultValue: "Loading…" })}
        </p>
      )}
      {error && (
        <p data-testid="settings-error" className="text-sm text-app-danger">
          {t("settings.error", { defaultValue: "Failed to load settings. Using local cache." })}
        </p>
      )}

      <Section title={t("settings.appearance", { defaultValue: "Appearance" })}>
        <Field label={t("settings.theme", { defaultValue: "Theme" })}>
          <Segmented
            testid="settings-theme"
            value={settings.theme}
            options={[
              { value: "light", label: t("theme.light", { defaultValue: "Light" }) },
              { value: "dark", label: t("theme.dark", { defaultValue: "Dark" }) },
              { value: "system", label: t("theme.system", { defaultValue: "System" }) },
            ]}
            onChange={(v) => {
              setTheme(v as Theme);
              update({ theme: v as Theme });
            }}
          />
        </Field>
        <Field label={t("settings.density", { defaultValue: "Density" })}>
          <Segmented
            testid="settings-density"
            value={settings.density}
            options={[
              { value: "comfortable", label: t("density.comfortable", { defaultValue: "Comfortable" }) },
              { value: "compact", label: t("density.compact", { defaultValue: "Compact" }) },
            ]}
            onChange={(v) => update({ density: v as Density })}
          />
        </Field>
        <Field label={t("settings.fontScale", { defaultValue: "Font size" })}>
          <Segmented
            testid="settings-font-scale"
            value={settings.fontScale}
            options={[
              { value: "sm", label: "Sm" },
              { value: "md", label: "Md" },
              { value: "lg", label: "Lg" },
            ]}
            onChange={(v) => update({ fontScale: v as FontScale })}
          />
        </Field>
        <Toggle
          testid="settings-reduced-motion"
          label={t("settings.reducedMotion", { defaultValue: "Reduce motion" })}
          value={settings.reducedMotion}
          onChange={(b) => update({ reducedMotion: b })}
        />
        <Toggle
          testid="settings-rtl"
          label={t("settings.rtl", { defaultValue: "Right-to-left layout" })}
          value={settings.rtl}
          onChange={(b) => update({ rtl: b })}
        />
      </Section>

      <Section title={t("settings.behavior", { defaultValue: "Behavior" })}>
        <Field label={t("settings.defaultRoot", { defaultValue: "Default root path" })}>
          <input
            data-testid="settings-default-root"
            value={settings.defaultRoot}
            onChange={(e) => update({ defaultRoot: e.target.value })}
            className="h-9 w-64 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          />
        </Field>
      </Section>

      {mutation.isError && (
        <p data-testid="settings-save-error" className="text-sm text-app-danger">
          {t("settings.saveError", { defaultValue: "Failed to save. Changes may be reverted on reload." })}
        </p>
      )}
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

function Toggle({
  testid,
  label,
  value,
  onChange,
}: {
  testid: string;
  label: string;
  value: boolean;
  onChange: (b: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-3 text-sm text-app-foreground">
      <span className="text-app-muted-foreground">{label}</span>
      <button
        type="button"
        data-testid={testid}
        role="switch"
        aria-checked={value}
        onClick={() => onChange(!value)}
        className={[
          "relative inline-flex h-6 w-11 items-center rounded-pill transition-colors",
          value ? "bg-app-primary" : "bg-app-surface-muted",
        ].join(" ")}
      >
        <span
          aria-hidden
          className={[
            "inline-block h-4 w-4 rounded-pill bg-white shadow transition-transform",
            value ? "translate-x-6" : "translate-x-1",
          ].join(" ")}
        />
      </button>
    </label>
  );
}
