import { useState } from "react";
import type { ZoneCapability, ZoneState } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { lightingApi } from "./api";

type Props = {
  controllerId: string;
  zone: ZoneCapability;
  desired?: ZoneState;
  maxBrightness?: number;
  disabled: boolean;
  detail?: boolean;
  act: (action: () => Promise<unknown>, success: string) => Promise<void>;
};

// A draft belongs to one zone. Refreshes keep unsaved edits; the backend remains
// authoritative for capabilities, ownership and the current maximum brightness.
export function ManualZoneEditor({ controllerId, zone, desired, maxBrightness, disabled, detail, act }: Props) {
  const { t } = useTranslation();
  const initial = desired?.desiredMode;
  const saved = initial?.case === "staticColor" || initial?.case === "effect" ? initial.value : undefined;
  const initialColor = saved?.color ? `#${[saved.color.red, saved.color.green, saved.color.blue].map((value) => value.toString(16).padStart(2, "0")).join("")}` : "#57be8a";
  const [color, setColor] = useState(initialColor);
  const [brightness, setBrightness] = useState(saved?.brightnessPercent ?? 20);
  const initialMode = initial?.case === "effect" ? `effect:${initial.value.effectId}` : zone.supportsStaticColor ? "static" : zone.supportedEffects.length ? `effect:${zone.supportedEffects[0]}` : "static";
  const [mode, setMode] = useState(initialMode);
  const effects = [...new Set(zone.supportedEffects)];
  const effectId = mode.startsWith("effect:") ? mode.slice(7) : undefined;
  const supported = effectId !== undefined ? effects.includes(effectId) : zone.supportsStaticColor;
  const cap = Math.max(0, Math.min(100, maxBrightness ?? 0));
  const outputBrightness = Math.min(brightness, cap);
  const apply = () => {
    if (disabled || !supported || maxBrightness === undefined) return;
    const rgb = { red: parseInt(color.slice(1, 3), 16), green: parseInt(color.slice(3, 5), 16), blue: parseInt(color.slice(5, 7), 16) };
    void act(() => effectId !== undefined
      ? lightingApi.setEffect(controllerId, zone.zoneId, effectId, rgb, outputBrightness)
      : lightingApi.setStatic(controllerId, zone.zoneId, rgb, outputBrightness), t(strings.lightDesk.controllerUi.colorSaved));
  };
  return <div className="lightdesk-zone-controls">
    {detail ? <div className="lightdesk-preview-strip" aria-label={t(strings.lightDesk.controllerUi.illustrativePreview)}><i style={{ background: color }}/><i style={{ background: color }}/><i style={{ background: color }}/><i style={{ background: color }}/><span>{t(strings.lightDesk.controllerUi.previewOnly)}</span></div> : null}
    <label>{t(strings.lightDesk.manualUi.mode)}
      <select aria-label={`${t(strings.lightDesk.manualUi.mode)} ${zone.name}`} value={mode} onChange={(event) => setMode(event.target.value)}>
        {zone.supportsStaticColor ? <option value="static">{t(strings.lightDesk.staticColor)}</option> : null}
        {effects.map((effect) => <option key={effect} value={`effect:${effect}`}>{effect}</option>)}
        {!supported ? <option value={mode}>{effectId ?? t(strings.lightDesk.staticColor)} · {t(strings.lightDesk.unsupported)}</option> : null}
      </select>
    </label>
    <label>{t(strings.lightDesk.color)}<input type="color" aria-label={`${t(strings.lightDesk.color)} ${zone.name}`} value={color} onChange={(event) => setColor(event.target.value)}/></label>
    {detail ? <code>{color.toUpperCase()}</code> : null}
    <label>{t(strings.lightDesk.brightness)} {outputBrightness}%<input type="range" min="0" max={cap} value={outputBrightness} aria-label={`${t(strings.lightDesk.brightness)} ${zone.name}`} onChange={(event) => setBrightness(Number(event.target.value))}/></label>
    <p className="lightdesk-meta">{maxBrightness === undefined ? t(strings.lightDesk.policyUnavailable) : `${t(strings.lightDesk.presetsUi.maximumBrightness)} · ${cap}%`}</p>
    {!supported ? <p className="lightdesk-warning">{t(strings.lightDesk.manualUi.unsupported)}</p> : null}
    <p className="lightdesk-muted">{t(strings.lightDesk.manualUi.ownership)}</p>
    <button className="lightdesk-button lightdesk-button--primary" aria-label={detail ? `${t(strings.lightDesk.controllerUi.saveApply)} · ${zone.name}` : undefined} disabled={disabled || !supported || maxBrightness === undefined} onClick={apply}>{t(detail || effectId !== undefined ? strings.lightDesk.controllerUi.saveApply : strings.lightDesk.applyStatic)}</button>
    {detail ? <button className="lightdesk-button" onClick={() => { setColor(initialColor); setBrightness(saved?.brightnessPercent ?? 20); setMode(initialMode); }}>{t(strings.lightDesk.presetsUi.cancelEdits)}</button> : null}
  </div>;
}
