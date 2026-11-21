import { CachedModeReadback, OperationState, type ZoneApplyResult, type ZoneState } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { colorHex, rgbLabel, scaledChannels } from "./colorEvidence";

type DesiredMode = ZoneState["desiredMode"];

export function EffectDeliveryNotice({ desired, lastApply }: { desired?: ZoneState; lastApply?: ZoneApplyResult }) {
  const { t } = useTranslation();
  const requested = desired?.desiredMode;
  const applied = lastApply?.desired?.desiredMode;
  if (requested?.case !== "effect" || !applied) return null;

  const confirmed = lastApply.operation === OperationState.BACKEND_ACKNOWLEDGED && lastApply.backendAcknowledged;
  if (applied.case === "staticColor") {
    return <p className="lightdesk-warning lightdesk-effect-delivery" role={confirmed ? "status" : "alert"}>
      {t(confirmed ? strings.lightDesk.effectStaticFallback : strings.lightDesk.effectStaticUnconfirmed, { effect: requested.value.effectId, percent: applied.value.brightnessPercent })}
    </p>;
  }
  if (applied.case === "off") {
    return <p className="lightdesk-warning lightdesk-effect-delivery" role={confirmed ? "status" : "alert"}>
      {t(confirmed ? strings.lightDesk.effectOffFallback : strings.lightDesk.effectOffUnconfirmed, { effect: requested.value.effectId })}
    </p>;
  }
  if (applied.case === "effect") {
    const readbackMatches = lastApply.cachedModeReadback === CachedModeReadback.MATCHES;
    return <p className="lightdesk-warning lightdesk-effect-delivery" role={confirmed && readbackMatches ? "status" : "alert"}>
      {t(confirmed && readbackMatches ? strings.lightDesk.effectNativeModeConfirmed : strings.lightDesk.effectNativeModeUnverified, { effect: requested.value.effectId })}
    </p>;
  }
  return null;
}

export function ZoneColorEvidence({ title, mode, scaledLabel, offLabel, noColorLabel, missingLabel }: { title: string; mode?: DesiredMode; scaledLabel: string; offLabel: string; noColorLabel: string; missingLabel: string }) {
  const { t } = useTranslation();
  if (!mode || mode.case === undefined) return <div className="lightdesk-color-evidence"><strong>{title}</strong><span>{missingLabel}</span></div>;
  if (mode.case === "off") return <div className="lightdesk-color-evidence"><strong>{title}</strong><span>{offLabel}</span></div>;

  const color = mode.value.color;
  if (!color) return <div className="lightdesk-color-evidence"><strong>{title}</strong><span>{noColorLabel}</span></div>;

  const hex = colorHex(color);
  if (!hex) return <div className="lightdesk-color-evidence"><strong>{title}</strong><span>{noColorLabel}</span></div>;
  const brightness = mode.value.brightnessPercent;
  const isStatic = mode.case === "staticColor";
  const scaled = isStatic ? scaledChannels(color, brightness) : undefined;
  const scaledHex = scaled ? colorHex(scaled) : undefined;

  return <div className="lightdesk-color-evidence">
    <strong>{title}</strong>
    <div className="lightdesk-color-values">
      <span className="lightdesk-color-sample" style={{ backgroundColor: hex }} aria-hidden="true" />
      <span><b>{hex}</b><small>{t(strings.lightDesk.rgbChannels, { channels: rgbLabel(color) })} · {t(strings.lightDesk.percentageValue, { percent: brightness })}</small></span>
    </div>
    {mode.case === "effect" ? <small className="lightdesk-muted">{t(strings.lightDesk.effectColorInput)}</small> : null}
    {scaled && scaledHex ? <div className="lightdesk-scaled-color">
      <span className="lightdesk-color-sample lightdesk-color-sample--scaled" style={{ backgroundColor: scaledHex }} aria-hidden="true" />
      <span><small>{scaledLabel}</small><b>{scaledHex}</b><small>{t(strings.lightDesk.rgbChannels, { channels: rgbLabel(scaled) })}</small></span>
    </div> : null}
  </div>;
}
