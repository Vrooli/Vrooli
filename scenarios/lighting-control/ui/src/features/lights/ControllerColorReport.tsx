import { CircleHelp } from "lucide-react";
import type { Controller, ZoneStatus } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { colorHex, rgbLabel } from "./colorEvidence";

export function ControllerColorReport({ controller, statuses, compact = false }: { controller: Controller; statuses: ZoneStatus[]; compact?: boolean }) {
  const { t } = useTranslation();
  const colors = controller.reportedLedColors;
  const observedAt = timestampMillis(controller.reportedColorsObservedAt);
  const latestAttempt = statuses.reduce<number | undefined>((latest, status) => {
    const completed = timestampMillis(status.lastApply?.completedAt);
    return completed === undefined ? latest : Math.max(latest ?? completed, completed);
  }, undefined);
  const reportRelation = observedAt === undefined || latestAttempt === undefined ? undefined : observedAt < latestAttempt ? strings.lightDesk.reportedColorPredatesCommand : strings.lightDesk.reportedColorAfterCommand;
  const source = controller.reportedColorSource === "openrgb_sdk_controller_data_cached_state"
    ? strings.lightDesk.reportedColorSourceOpenRgbCache
    : controller.reportedColorSource ? strings.lightDesk.reportedColorSourceUnknown : undefined;

  return <section className={`lightdesk-reported-colors${compact ? " lightdesk-reported-colors--compact" : ""}`} aria-labelledby={`reported-color-heading-${controller.controllerId}`}>
    <h3 id={`reported-color-heading-${controller.controllerId}`}>{t(strings.lightDesk.openRgbReportedState)}</h3>
    {controller.reportedActiveMode ? <p className="lightdesk-muted">{t(strings.lightDesk.controllerUi.reportedMode)}: <strong>{controller.reportedActiveMode}</strong></p> : null}
    {controller.reportedActiveModeObservedAt ? <p className="lightdesk-meta">{t(strings.lightDesk.reportedModeTime, { time: formatTimestamp(controller.reportedActiveModeObservedAt) })}</p> : null}
    {source ? <p className="lightdesk-meta">{t(strings.lightDesk.reportedColorSource, { source: t(source) })}</p> : null}
    {controller.reportedColorsObservedAt ? <p className="lightdesk-meta">{t(strings.lightDesk.reportedColorTime, { time: formatTimestamp(controller.reportedColorsObservedAt) })}</p> : null}
    {colors.length ? <ul className="lightdesk-reported-color-list">{colors.slice(0, 16).map((reported, index) => {
      const color = reported.color;
      const hex = colorHex(color);
      return <li key={reported.index ?? index}>
        {color && hex ? <span className="lightdesk-color-sample" style={{ backgroundColor: hex }} aria-hidden="true"/> : <span className="lightdesk-color-sample lightdesk-color-sample--unknown" aria-hidden="true"/>}
        <span><strong>{reported.displayName || t(strings.lightDesk.unknownLed, { index: reported.index ?? index + 1 })}</strong><small>{color && hex ? `${hex} · ${t(strings.lightDesk.rgbChannels, { channels: rgbLabel(color) })}` : t(strings.lightDesk.noColorValue)}</small></span>
      </li>;
    })}{colors.length > 16 ? <li className="lightdesk-meta">+{colors.length - 16}</li> : null}</ul> : <p className="lightdesk-muted">{t(strings.lightDesk.reportedColorUnavailable)}</p>}
    {reportRelation ? <p className="lightdesk-meta">{t(reportRelation)}</p> : null}
    <p className="lightdesk-warning"><CircleHelp aria-hidden="true"/>{t(strings.lightDesk.reportedColorScope)}</p>
  </section>;
}

function timestampMillis(timestamp?: { seconds: bigint; nanos: number }): number | undefined {
  if (!timestamp) return undefined;
  const seconds = Number(timestamp.seconds);
  return Number.isFinite(seconds) ? seconds * 1000 + timestamp.nanos / 1_000_000 : undefined;
}

function formatTimestamp(timestamp: { seconds: bigint; nanos: number }): string {
  const milliseconds = timestampMillis(timestamp);
  return milliseconds === undefined ? "" : new Date(milliseconds).toLocaleString();
}
