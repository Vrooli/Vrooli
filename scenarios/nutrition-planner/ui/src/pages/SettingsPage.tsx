import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Link } from "react-router-dom";
import { useEffect, useState } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation, type Locale } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { fetchDiagnostics, type DiagnosticsReport } from "../api/diagnostics";
import { ensureWorkspace } from "../api/workspace";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
// Literal references so the strings lint can see every catalog key in use.
const THEME_LABEL_KEY: Record<ThemeChoice, (typeof strings.theme.choice)[ThemeChoice]> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

/**
 * Settings owns every preference; nothing else in the shell duplicates one.
 * Appearance and language are the two every scenario has. Add the rest of
 * yours as rows in the same list, grouped by what they change.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const [diagnostics, setDiagnostics] = useState<DiagnosticsReport>();
  const [diagnosticsError, setDiagnosticsError] = useState("");

  useEffect(() => {
    let active = true;
    void ensureWorkspace().then((workspace) => fetchDiagnostics(workspace.id)).then((report) => {
      if (active) setDiagnostics(report);
    }).catch((error: unknown) => {
      if (active) setDiagnosticsError(error instanceof Error ? error.message : "Unable to load data health.");
    });
    return () => { active = false; };
  }, []);

  return (
    <section data-testid={selectors.pages.settings} aria-labelledby="settings-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="settings-heading" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)} />
      <SettingsList variant="auto">
        <SettingsList.Group label={t(strings.pages.settings.preferences)}>
          <SettingsList.Row label={t(strings.pages.settings.themeHeading)} hint={t(strings.pages.settings.themeHint)}>
            <Select
              aria-label={t(strings.theme.switcherLabel)}
              data-testid={selectors.settingsPage.themeSelect}
              value={choice}
              onChange={(event) => setTheme(event.target.value as ThemeChoice)}
              options={THEME_CHOICES.map((c) => ({ value: c, label: t(THEME_LABEL_KEY[c]) }))}
            />
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.localeHeading)} hint={t(strings.pages.settings.localeHint)}>
            <Select
              aria-label={t(strings.locale.switcherLabel)}
              data-testid={selectors.settingsPage.localeSelect}
              value={currentLocale}
              onChange={(event) => void setLocale(event.target.value as Locale)}
              options={SUPPORTED_LOCALES.map((lng) => ({ value: lng, label: getLocaleConfig(lng).nativeLabel }))}
            />
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <section aria-labelledby="data-health-heading" className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h2 id="data-health-heading" className="text-lg font-semibold text-slate-900">Data health</h2>
        <p className="mt-1 text-sm text-slate-600">Actionable gaps for this workspace. Unknown data is reported honestly, not scored as failure.</p>
        {diagnosticsError && <p role="alert" className="mt-3 text-sm text-red-700">{diagnosticsError}</p>}
        {!diagnostics && !diagnosticsError && <p role="status" className="mt-3 text-sm text-slate-600">Checking workspace data…</p>}
        {diagnostics && <>
          <p className="mt-3 text-sm text-slate-700">Database: {diagnostics.databaseOk ? "available" : "unavailable"} · Schema version: {diagnostics.schemaVersion}</p>
          {diagnostics.findings.length === 0 ? <p className="mt-3 text-sm text-emerald-700">No material data-health findings for the current workspace.</p> : <ul className="mt-3 space-y-3" aria-label="Data health findings">
            {diagnostics.findings.map((finding) => <li key={finding.code} className="rounded border border-amber-200 bg-amber-50 p-3 text-sm"><p className="font-medium text-amber-950">{finding.message} ({finding.count})</p><p className="mt-1 text-amber-900">{finding.action}</p></li>)}
          </ul>}
          <details className="mt-4"><summary className="cursor-pointer text-sm font-medium text-slate-700">Provider availability</summary><ul className="mt-2 space-y-1 text-sm text-slate-600">{diagnostics.providers.map((provider) => <li key={provider.name}><span className="font-medium">{provider.name}</span>: {provider.state} — {provider.reason}</li>)}</ul></details>
        </>}
      </section>
      <Link className="min-h-11 rounded border px-4 py-3 text-sm font-medium text-slate-800" to="/transfer">Open backup and restore</Link>
    </section>
  );
}
