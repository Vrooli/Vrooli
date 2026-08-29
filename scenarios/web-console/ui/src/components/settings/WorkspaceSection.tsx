import { LayoutGrid, LayoutList, PanelLeft, PanelRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useWakeLockStatus } from "../../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import { SettingsSlider, SettingsToggle } from "./primitives";
import LocaleSwitcher from "../LocaleSwitcher";
import ToolbarCustomizer from "./ToolbarCustomizer";
import { deviceIdentity, setDeviceLabel } from "../../lib/deviceIdentity";
import { useSecureContextCapabilities } from "../../hooks/useSecureContextCapabilities";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/0.1.5";

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
  const handedness = useWorkspaceStore((state) => state.handedness);
  const setHandedness = useWorkspaceStore((state) => state.setHandedness);
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
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.workspaceSection.eyebrow)}
        title={t(strings.settings.workspaceSection.title)}
        description={t(strings.settings.workspaceSection.description)}
      />

      <SettingsList.Group>
        {unavailableCapabilities && (
          <p className="text-xs text-wc-text-secondary" role="status" data-testid="secure-context-capabilities">
            Browser capabilities unavailable in this context: {unavailableCapabilities}.
          </p>
        )}
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.paneLayoutLabel)}
          hint={t(strings.settings.workspaceSection.paneLayoutHint)} control="compact">{(
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
          )}</SettingsList.Row>

        {/* Which edge drawers open from, and therefore which way their
            gestures run. This is a reach preference, not a language one: it
            moves the drawer without mirroring any text. Changing the interface
            language is what mirrors text, and it is applied on top of this. */}
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.handednessLabel)}
          hint={t(strings.settings.workspaceSection.handednessHint)} control="compact">{(
            <div className="flex gap-1">
              <Button
                data-testid="handedness-inline-start"
                variant={handedness === "inline-start" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                aria-pressed={handedness === "inline-start"}
                onClick={() => { setHandedness("inline-start"); }}
              >
                <PanelLeft className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.handednessStart)}
              </Button>
              <Button
                data-testid="handedness-inline-end"
                variant={handedness === "inline-end" ? "default" : "outline"}
                size="sm"
                className="h-8 px-3"
                aria-pressed={handedness === "inline-end"}
                onClick={() => { setHandedness("inline-end"); }}
              >
                <PanelRight className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.handednessEnd)}
              </Button>
            </div>
          )}</SettingsList.Row>

        {/* The toolbar has three independent choices — which controls, how
            large, how many rows — so it gets a block of its own rather than a
            SettingsList.Row's single control slot. */}
        <div className="border-t border-wc-default pt-4">
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
          <SettingsList.Row
            label={t(strings.settings.workspaceSection.minimapLabel)}
            hint={t(strings.settings.workspaceSection.minimapHint)} control="compact">{(
              <SettingsToggle
                testId="minimap-toggle"
                checked={isMinimapVisible}
                onCheckedChange={setMinimapVisible}
              />
            )}</SettingsList.Row>
        )}
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.adaptiveChromeLabel)}
          hint={t(strings.settings.workspaceSection.adaptiveChromeHint)} control="compact">{(
            <SettingsToggle
              testId="adaptive-chrome-toggle"
              checked={adaptiveChrome}
              onCheckedChange={setAdaptiveChrome}
            />
          )}</SettingsList.Row>
        <SettingsList.Row
          label="Touch scroll sensitivity"
          hint="Adjust finger and trackpad scrolling independently." control="wide">{(
            <SettingsSlider
              testId="touch-scroll-sensitivity"
              value={touchScrollSensitivity}
              onCommit={setTouchScrollSensitivity}
              min={0.1}
              max={4}
              step={0.1}
              defaultMarker={1}
              formatValue={(value) => value.toFixed(1)}
            />
          )}</SettingsList.Row>
        <SettingsList.Row
          label="Wheel scroll sensitivity"
          hint="Adjust mouse-wheel scrolling independently." control="wide">{(
            <SettingsSlider
              testId="wheel-scroll-sensitivity"
              value={wheelScrollSensitivity}
              onCommit={setWheelScrollSensitivity}
              min={0.1}
              max={4}
              step={0.1}
              defaultMarker={1}
              formatValue={(value) => value.toFixed(1)}
            />
          )}</SettingsList.Row>
        <div className="flex justify-end">
          <Button type="button" variant="outline" size="sm" onClick={resetScrollSensitivities}>
            Reset scroll sensitivities
          </Button>
        </div>
        <SettingsList.Row
          label="tmux mouse mode for new persistent panes"
          hint="When enabled, new persistent panes let tmux capture mouse scrolling. Existing panes keep their current setting." control="compact">{(
            <SettingsToggle
              testId="tmux-mouse-mode-default-toggle"
              checked={tmuxMouseMode}
              onCheckedChange={setTmuxMouseMode}
            />
          )}</SettingsList.Row>
        <SettingsList.Row
          label="Prediction latency threshold"
          hint="Underline speculative characters only when round-trip latency exceeds this value." control="wide">{(
            <SettingsSlider
              testId="prediction-latency-threshold"
              value={predictionLatencyThresholdMs}
              onCommit={setPredictionLatencyThresholdMs}
              min={0}
              max={1000}
              step={5}
              formatValue={(value) => `${String(value)} ms`}
            />
          )}</SettingsList.Row>
        <SettingsList.Row
		  label={t(strings.deviceIdentity.label)}
		  hint={t(strings.deviceIdentity.hint)} control="compact">{<input className="h-8 rounded border border-wc-default bg-wc-surface-input px-2 text-sm" value={deviceLabel} onChange={(event) => { setLocalDeviceLabel(event.target.value); setDeviceLabel(event.target.value); }} />}</SettingsList.Row>
		<SettingsList.Row
          label={t(strings.settings.workspaceSection.localeLabel)}
          hint={t(strings.settings.workspaceSection.localeHint)} control="compact">{<LocaleSwitcher />}</SettingsList.Row>
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.keepAwakeLabel)}
          hint={<span className={wakeLockHintClass}>{wakeLockHint}</span>} control="compact">{(
            <SettingsToggle
              testId="keep-screen-awake-toggle"
              checked={keepScreenAwake}
              onCheckedChange={setKeepScreenAwake}
            />
          )}</SettingsList.Row>
      </SettingsList.Group>
    </SettingsList>
  );
}
