import "../settings.css";
import { useEffect, useState } from "react";
import { Check, Download, RotateCcw, ShieldCheck } from "lucide-react";

import { type Theme } from "../../components/theme/theme-context";
import { useTheme } from "../../components/theme/useTheme";
import {
  SUPPORTED_LOCALES,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
  useTranslation,
} from "../../i18n";
import { readWorkbenchPreferences, writeWorkbenchPreferences, type WorkbenchPreferences } from "../preferences";
import {
  Banner,
  Chip,
  Select,
  Toggle,
  WbButton,
} from "../primitives";

export function SettingsPage() {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();
  const locale = getCurrentLocale();
  const [preferences, setPreferences] = useState<WorkbenchPreferences>(() => readWorkbenchPreferences());
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    writeWorkbenchPreferences(preferences);
  }, [preferences]);

  const update = <K extends keyof WorkbenchPreferences>(key: K, value: WorkbenchPreferences[K]) => {
    setPreferences((current) => ({ ...current, [key]: value }));
    setSaved(true);
    window.setTimeout(() => setSaved(false), 1600);
  };

  const exportPreferences = () => {
    const payload = {
      theme,
      locale,
      workbench: preferences,
      exportedAt: new Date().toISOString(),
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "rcl-workbench-preferences.json";
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const reset = () => {
    for (const key of Object.keys(window.localStorage)) {
      if (key.startsWith("rcl.workbench.")) window.localStorage.removeItem(key);
    }
    setPreferences(readWorkbenchPreferences());
    setTheme("system");
    setSaved(true);
    window.setTimeout(() => setSaved(false), 1600);
  };

  return (
    <div className="wb-page" data-testid="settings-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.settings.kicker", { defaultValue: "Library · Settings" })}</p>
          <h1 className="wb-h1">{t("workbench.settings.title", { defaultValue: "Workspace preferences" })}</h1>
          <p className="wb-lede">
            {t("workbench.settings.lede", {
              defaultValue:
                "These settings are local to this browser. Library policy, evidence thresholds and release rules stay in the scenario configuration.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          {saved ? <Chip tone="green">{t("workbench.settings.saved", { defaultValue: "Saved locally" })}</Chip> : null}
          <WbButton variant="ghost" size="md" onClick={reset}>
            <RotateCcw aria-hidden size={14} />
            {t("workbench.settings.reset", { defaultValue: "Reset to defaults" })}
          </WbButton>
        </div>
      </div>

      <div className="wb-settings-grid">
        <section className="wb-settings-card">
          <header>{t("workbench.settings.appearance", { defaultValue: "Appearance" })}</header>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.theme", { defaultValue: "Preview theme" })}</b>
              <small>
                {t("workbench.settings.themeBody", {
                  defaultValue:
                    "Nocturne chrome stays dark. This preference is read by preview surfaces that opt into the shared theme.",
                })}
              </small>
            </div>
            <Select
              aria-label={t("workbench.settings.theme", { defaultValue: "Preview theme" })}
              value={theme}
              onChange={(event) => setTheme(event.target.value as Theme)}
              style={{ width: "auto", minWidth: 120 }}
            >
              <option value="system">{t("theme.system", { defaultValue: "System" })}</option>
              <option value="dark">{t("theme.dark", { defaultValue: "Dark" })}</option>
              <option value="light">{t("theme.light", { defaultValue: "Light" })}</option>
            </Select>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.density", { defaultValue: "Density" })}</b>
              <small>
                {t("workbench.settings.densityBody", {
                  defaultValue: "Compact leaves more room around fixtures and evidence rows.",
                })}
              </small>
            </div>
            <Select
              aria-label={t("workbench.settings.density", { defaultValue: "Density" })}
              value={preferences.density}
              onChange={(event) => update("density", event.target.value as WorkbenchPreferences["density"])}
              style={{ width: "auto", minWidth: 130 }}
            >
              <option value="comfortable">{t("workbench.settings.comfortable", { defaultValue: "Comfortable" })}</option>
              <option value="compact">{t("workbench.settings.compact", { defaultValue: "Compact" })}</option>
            </Select>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.defaultDevice", { defaultValue: "Default device" })}</b>
              <small>
                {t("workbench.settings.defaultDeviceBody", {
                  defaultValue: "Used when a component or page opens in context.",
                })}
              </small>
            </div>
            <Select
              aria-label={t("workbench.settings.defaultDevice", { defaultValue: "Default device" })}
              value={preferences.defaultDevice}
              onChange={(event) => update("defaultDevice", event.target.value as WorkbenchPreferences["defaultDevice"])}
              style={{ width: "auto", minWidth: 130 }}
            >
              <option value="desktop">{t("workbench.settings.desktop", { defaultValue: "Desktop" })}</option>
              <option value="tablet">{t("workbench.settings.tablet", { defaultValue: "Tablet" })}</option>
              <option value="phone">{t("workbench.settings.phone", { defaultValue: "Phone" })}</option>
            </Select>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.reducedMotion", { defaultValue: "Reduce motion" })}</b>
              <small>
                {t("workbench.settings.reducedMotionBody", {
                  defaultValue: "The workbench always respects the operating system setting. This forces it on.",
                })}
              </small>
            </div>
            <Toggle
              checked={preferences.reduceMotion}
              onChange={(next) => update("reduceMotion", next)}
              label={t("workbench.settings.reducedMotion", { defaultValue: "Reduce motion" })}
            />
          </div>
        </section>

        <section className="wb-settings-card">
          <header>{t("workbench.settings.language", { defaultValue: "Language" })}</header>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.interfaceLanguage", { defaultValue: "Interface language" })}</b>
              <small>
                {t("workbench.settings.interfaceLanguageBody", {
                  defaultValue: "Story text and source are never translated.",
                })}
              </small>
            </div>
            <Select
              aria-label={t("workbench.settings.interfaceLanguage", { defaultValue: "Interface language" })}
              value={locale}
              onChange={(event) => void setLocale(event.target.value as (typeof SUPPORTED_LOCALES)[number])}
              style={{ width: "auto", minWidth: 140 }}
            >
              {SUPPORTED_LOCALES.map((code) => (
                <option key={code} value={code}>
                  {getLocaleConfig(code).nativeLabel}
                </option>
              ))}
            </Select>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.localeNote", { defaultValue: "Translation coverage" })}</b>
              <small>
                {t("workbench.settings.localeNoteBody", {
                  defaultValue:
                    "The workbench shell is translated. Some evidence text comes from the API and remains in its source language.",
                })}
              </small>
            </div>
            <Chip tone="ice">{locale}</Chip>
          </div>
        </section>

        <section className="wb-settings-card">
          <header>{t("workbench.settings.evidence", { defaultValue: "Evidence defaults" })}</header>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.freshness", { defaultValue: "Freshness window" })}</b>
              <small>
                {t("workbench.settings.freshnessBody", {
                  defaultValue: "Evidence older than this is labelled stale when a capture timestamp exists.",
                })}
              </small>
            </div>
            <Select
              aria-label={t("workbench.settings.freshness", { defaultValue: "Freshness window" })}
              value={preferences.freshness}
              onChange={(event) => update("freshness", event.target.value as WorkbenchPreferences["freshness"])}
              style={{ width: "auto", minWidth: 120 }}
            >
              <option value="24h">{t("workbench.settings.hours24", { defaultValue: "24 hours" })}</option>
              <option value="7d">{t("workbench.settings.days7", { defaultValue: "7 days" })}</option>
              <option value="30d">{t("workbench.settings.days30", { defaultValue: "30 days" })}</option>
            </Select>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.captureViewports", { defaultValue: "Capture viewports" })}</b>
              <small>
                {t("workbench.settings.captureViewportsBody", {
                  defaultValue: "Desktop and phone are required for a production-ready claim.",
                })}
              </small>
            </div>
            <div className="wb-row">
              <Chip tone={preferences.captureDesktop ? "green" : "amber"}>{t("workbench.settings.desktop", { defaultValue: "Desktop" })}</Chip>
              <Chip tone={preferences.capturePhone ? "green" : "amber"}>{t("workbench.settings.phone", { defaultValue: "Phone" })}</Chip>
            </div>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.runOnPublish", { defaultValue: "Run evidence on publish" })}</b>
              <small>
                {t("workbench.settings.runOnPublishBody", {
                  defaultValue: "Captures the selected viewports after a release succeeds.",
                })}
              </small>
            </div>
            <Toggle
              checked={preferences.runOnPublish}
              onChange={(next) => update("runOnPublish", next)}
              label={t("workbench.settings.runOnPublish", { defaultValue: "Run evidence on publish" })}
            />
          </div>
        </section>

        <section className="wb-settings-card">
          <header>{t("workbench.settings.localData", { defaultValue: "Local data" })}</header>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.export", { defaultValue: "Export preferences" })}</b>
              <small>
                {t("workbench.settings.exportBody", {
                  defaultValue: "Download this browser's workbench settings as JSON.",
                })}
              </small>
            </div>
            <WbButton variant="ghost" size="sm" onClick={exportPreferences}>
              <Download aria-hidden size={13} />
              {t("workbench.settings.exportAction", { defaultValue: "Export" })}
            </WbButton>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.resetLocal", { defaultValue: "Reset local data" })}</b>
              <small>
                {t("workbench.settings.resetLocalBody", {
                  defaultValue: "Clears workbench preferences and local design drafts. Published library data is untouched.",
                })}
              </small>
            </div>
            <WbButton variant="danger" size="sm" onClick={reset}>
              <RotateCcw aria-hidden size={13} />
              {t("workbench.settings.resetAction", { defaultValue: "Reset" })}
            </WbButton>
          </div>
          <div className="wb-setting-row">
            <div className="wb-setting-copy">
              <b>{t("workbench.settings.boundary", { defaultValue: "Local vs scenario policy" })}</b>
              <small>
                {t("workbench.settings.boundaryBody", {
                  defaultValue:
                    "Nothing on this page changes library policy, evidence thresholds or release rules.",
                })}
              </small>
            </div>
            <ShieldCheck aria-hidden size={16} style={{ color: "var(--wb-green)" }} />
          </div>
        </section>
      </div>

      <div style={{ height: 14 }} />
      <Banner tone="ice" icon={<Check aria-hidden size={15} />}>
        {t("workbench.settings.footer", {
          defaultValue:
            "Preferences are saved automatically in this browser. A design draft is local until it is accepted; a release is published only through the governed asset flow.",
        })}
      </Banner>
    </div>
  );
}
