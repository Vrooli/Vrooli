import { LayoutGrid, LayoutList, PanelLeft } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useWakeLockStatus } from "../../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";
import LocaleSwitcher from "../LocaleSwitcher";
import ToolbarCustomizer from "./ToolbarCustomizer";
import { deviceIdentity, setDeviceLabel } from "../../lib/deviceIdentity";
import { useSecureContextCapabilities } from "../../hooks/useSecureContextCapabilities";

const STATUS_HINT_KEYS = {
  active: strings.settings.workspaceSection.wakeLockActive,
  off: strings.settings.workspaceSection.wakeLockDefault,
  unsupported: strings.settings.workspaceSection.wakeLockUnsupported,
  denied: strings.settings.workspaceSection.wakeLockDenied,
  released: strings.settings.workspaceSection.wakeLockReleased,
} as const;

const STATUS_HINT_CLASSES: Record<string, string | undefined> = {
  active: "text-wc-accent",
  denied: "text-wc-error",
  released: "text-yellow-500",
};

export default function WorkspaceSection() {
  const { t } = useTranslation();
  const isMinimapVisible = useWorkspaceStore((state) => state.isMinimapVisible);
  const setMinimapVisible = useWorkspaceStore((state) => state.setMinimapVisible);
  const displayMode = useWorkspaceStore((state) => state.displayMode);
  const setDisplayMode = useWorkspaceStore((state) => state.setDisplayMode);
  const keepScreenAwake = useWorkspaceStore((state) => state.keepScreenAwake);
  const setKeepScreenAwake = useWorkspaceStore((state) => state.setKeepScreenAwake);
  const adaptiveChrome = useWorkspaceStore((state) => state.adaptiveChrome);
  const setAdaptiveChrome = useWorkspaceStore((state) => state.setAdaptiveChrome);
  const touchScrollSensitivity = useWorkspaceStore((state) => state.touchScrollSensitivity);
  const wheelScrollSensitivity = useWorkspaceStore((state) => state.wheelScrollSensitivity);
  const setTouchScrollSensitivity = useWorkspaceStore((state) => state.setTouchScrollSensitivity);
  const setWheelScrollSensitivity = useWorkspaceStore((state) => state.setWheelScrollSensitivity);
  const tmuxMouseMode = useWorkspaceStore((state) => state.tmuxMouseMode);
  const setTmuxMouseMode = useWorkspaceStore((state) => state.setTmuxMouseMode);
  const predictionLatencyThresholdMs = useWorkspaceStore((state) => state.predictionLatencyThresholdMs);
  const setPredictionLatencyThresholdMs = useWorkspaceStore((state) => state.setPredictionLatencyThresholdMs);
  const resetScrollSensitivities = useWorkspaceStore((state) => state.resetScrollSensitivities);
  const wakeLockStatus = useWakeLockStatus((s) => s.status);
  const capabilities = useSecureContextCapabilities();
  const unavailableCapabilities = Object.entries(capabilities)
    .filter(([, state]) => state === "unsupported")
    .map(([name]) => name)
    .join(", ");
  const [deviceLabel, setLocalDeviceLabel] = useState(() => deviceIdentity().label);

  const defaultHintKey = strings.settings.workspaceSection.wakeLockDefault;
  const wakeLockHint = keepScreenAwake
    ? t(STATUS_HINT_KEYS[wakeLockStatus] ?? defaultHintKey)
    : t(defaultHintKey);
  const wakeLockHintClass = keepScreenAwake
    ? STATUS_HINT_CLASSES[wakeLockStatus]
    : undefined;

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.workspaceSection.eyebrow)}
        title={t(strings.settings.workspaceSection.title)}
        description={t(strings.settings.workspaceSection.description)}
      />

      <SettingsCard className="space-y-4">
        {unavailableCapabilities && (
          <p className="text-xs text-wc-text-secondary" role="status" data-testid="secure-context-capabilities">
            Browser capabilities unavailable in this context: {unavailableCapabilities}.
          </p>
        )}
        <SettingsRow
          label={t(strings.settings.workspaceSection.paneLayoutLabel)}
          hint={t(strings.settings.workspaceSection.paneLayoutHint)}
          control={(
            <div className="flex items-center gap-2">
              <Button
                data-testid="display-mode-grid"
                variant={displayMode === "grid" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setDisplayMode("grid")}
              >
                <LayoutGrid className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.grid)}
              </Button>
              <Button
                data-testid="display-mode-tabs"
                variant={displayMode === "tabs" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setDisplayMode("tabs")}
              >
                <LayoutList className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.tabs)}
              </Button>
              <Button
                data-testid="display-mode-sidebar"
                variant={displayMode === "sidebar" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setDisplayMode("sidebar")}
              >
                <PanelLeft className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.sidebar)}
              </Button>
            </div>
          )}
        />

        {/* The toolbar has three independent choices — which controls, how
            large, how many rows — so it gets a block of its own rather than a
            SettingsRow's single control slot. */}
        <div className="space-y-2 border-t border-wc-default pt-4">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.mobileToolbarLabel)}
            </div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.workspaceSection.mobileToolbarHint)}
            </div>
          </div>
          <ToolbarCustomizer />
        </div>

        {displayMode === "grid" && (
          <SettingsRow
            label={t(strings.settings.workspaceSection.minimapLabel)}
            hint={t(strings.settings.workspaceSection.minimapHint)}
            control={(
              <SettingsToggle
                testId="minimap-toggle"
                checked={isMinimapVisible}
                onClick={() => setMinimapVisible(!isMinimapVisible)}
              />
            )}
          />
        )}
        <SettingsRow
          label={t(strings.settings.workspaceSection.adaptiveChromeLabel)}
          hint={t(strings.settings.workspaceSection.adaptiveChromeHint)}
          control={(
            <SettingsToggle
              testId="adaptive-chrome-toggle"
              checked={adaptiveChrome}
              onClick={() => setAdaptiveChrome(!adaptiveChrome)}
            />
          )}
        />
        <SettingsRow
          label="Touch scroll sensitivity"
          hint="Adjust finger and trackpad scrolling independently."
          control={(
            <label className="flex items-center gap-2 text-xs text-wc-text-secondary">
              <input aria-label="Touch scroll sensitivity" type="range" min="0.1" max="4" step="0.1" value={touchScrollSensitivity} onChange={(event) => setTouchScrollSensitivity(Number(event.target.value))} />
              {touchScrollSensitivity.toFixed(1)}
            </label>
          )}
        />
        <SettingsRow
          label="Wheel scroll sensitivity"
          hint="Adjust mouse-wheel scrolling independently."
          control={(
            <label className="flex items-center gap-2 text-xs text-wc-text-secondary">
              <input aria-label="Wheel scroll sensitivity" type="range" min="0.1" max="4" step="0.1" value={wheelScrollSensitivity} onChange={(event) => setWheelScrollSensitivity(Number(event.target.value))} />
              {wheelScrollSensitivity.toFixed(1)}
            </label>
          )}
        />
        <div className="flex justify-end">
          <Button type="button" variant="outline" size="sm" onClick={resetScrollSensitivities}>
            Reset scroll sensitivities
          </Button>
        </div>
        <SettingsRow
          label="tmux mouse mode for new persistent panes"
          hint="When enabled, new persistent panes let tmux capture mouse scrolling. Existing panes keep their current setting."
          control={(
            <SettingsToggle
              testId="tmux-mouse-mode-default-toggle"
              checked={tmuxMouseMode}
              onClick={() => setTmuxMouseMode(!tmuxMouseMode)}
            />
          )}
        />
        <SettingsRow
          label="Prediction latency threshold"
          hint="Underline speculative characters only when round-trip latency exceeds this value."
          control={(
            <label className="flex items-center gap-2 text-xs text-wc-text-secondary">
              <input aria-label="Prediction latency threshold" type="range" min="0" max="1000" step="5" value={predictionLatencyThresholdMs} onChange={(event) => setPredictionLatencyThresholdMs(Number(event.target.value))} />
              {predictionLatencyThresholdMs} ms
            </label>
          )}
        />
        <SettingsRow
		  label={t(strings.deviceIdentity.label)}
		  hint={t(strings.deviceIdentity.hint)}
		  control={<input className="h-8 rounded border border-wc-default bg-wc-surface-input px-2 text-sm" value={deviceLabel} onChange={(event) => { setLocalDeviceLabel(event.target.value); setDeviceLabel(event.target.value); }} />}
		/>
		<SettingsRow
          label={t(strings.settings.workspaceSection.localeLabel)}
          hint={t(strings.settings.workspaceSection.localeHint)}
          control={<LocaleSwitcher />}
        />
        <SettingsRow
          label={t(strings.settings.workspaceSection.keepAwakeLabel)}
          hint={wakeLockHint}
          hintClassName={wakeLockHintClass}
          control={(
            <SettingsToggle
              testId="keep-screen-awake-toggle"
              checked={keepScreenAwake}
              onClick={() => setKeepScreenAwake(!keepScreenAwake)}
            />
          )}
        />
      </SettingsCard>
    </div>
  );
}
