import { LayoutGrid, LayoutList, PanelLeft, PanelRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useWakeLockStatus } from "../../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import { SettingsSlider, SettingsToggle, SettingsIntro } from "./primitives";
import LocaleSwitcher from "../LocaleSwitcher";
import ToolbarCustomizer from "./ToolbarCustomizer";
import { deviceIdentity, setDeviceLabel } from "../../lib/deviceIdentity";
import { useSecureContextCapabilities } from "../../hooks/useSecureContextCapabilities";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";

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
    <SettingsList layout="aligned" variant="flush" density="compact" className="max-w-4xl">
      <SettingsIntro
        title={t(strings.settings.workspaceSection.title)}
        description={t(strings.settings.workspaceSection.description)}
      />

      <SettingsList.Group label={t(strings.settings.workspaceSection.eyebrow)}>
        {unavailableCapabilities && (
          <p className="text-xs text-wc-text-secondary" role="status" data-testid="secure-context-capabilities">
            Browser capabilities unavailable in this context: {unavailableCapabilities}.
          </p>
        )}
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.paneLayoutLabel)}
          hint={t(strings.settings.workspaceSection.paneLayoutHint)} control="compact">{(
            <ButtonGroup variant="segmented" collapse="never" label={t(strings.settings.workspaceSection.paneLayoutLabel)}>
              <Button
                data-testid="display-mode-grid"
                aria-pressed={displayMode === "grid"}
                variant="ghost"
                shape="square"
                size="lg"
                onClick={() => setDisplayMode("grid")}
              >
                <LayoutGrid className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.grid)}
              </Button>
              <Button
                data-testid="display-mode-tabs"
                aria-pressed={displayMode === "tabs"}
                variant="ghost"
                shape="square"
                size="lg"
                onClick={() => setDisplayMode("tabs")}
              >
                <LayoutList className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.tabs)}
              </Button>
              <Button
                data-testid="display-mode-sidebar"
                aria-pressed={displayMode === "sidebar"}
                variant="ghost"
                shape="square"
                size="lg"
                onClick={() => setDisplayMode("sidebar")}
              >
                <PanelLeft className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.sidebar)}
              </Button>
            </ButtonGroup>
          )}</SettingsList.Row>

        {/* Which edge drawers open from, and therefore which way their
            gestures run. This is a reach preference, not a language one: it
            moves the drawer without mirroring any text. Changing the interface
            language is what mirrors text, and it is applied on top of this. */}
        <SettingsList.Row
          label={t(strings.settings.workspaceSection.handednessLabel)}
          hint={t(strings.settings.workspaceSection.handednessHint)} control="compact">{(
            <ButtonGroup variant="segmented" collapse="never" label={t(strings.settings.workspaceSection.handednessLabel)}>
              <Button
                data-testid="handedness-inline-start"
                variant="ghost"
                shape="square"
                size="lg"
                aria-pressed={handedness === "inline-start"}
                onClick={() => { setHandedness("inline-start"); }}
              >
                <PanelLeft className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.handednessStart)}
              </Button>
              <Button
                data-testid="handedness-inline-end"
                variant="ghost"
                shape="square"
                size="lg"
                aria-pressed={handedness === "inline-end"}
                onClick={() => { setHandedness("inline-end"); }}
              >
                <PanelRight className="me-1 h-3.5 w-3.5" />
                {t(strings.settings.workspaceSection.handednessEnd)}
              </Button>
            </ButtonGroup>
          )}</SettingsList.Row>

        <SettingsList.Row
          label={t(strings.settings.workspaceSection.mobileToolbarLabel)}
          hint={t(strings.settings.workspaceSection.mobileToolbarHint)}
          control="block"
        >
          <ToolbarCustomizer />
        </SettingsList.Row>

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
      </SettingsList.Group>
      <SettingsList.Group label={t("settings.workspaceSection.appearanceAndScrolling", { defaultValue: "Appearance & scrolling" })}>
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
      </SettingsList.Group>
      <SettingsList.Group label={t("settings.workspaceSection.additionalOptions", { defaultValue: "Additional options" })}>
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
