import type { CSSProperties, ReactNode } from "react";
import { BatteryFull, Signal, Wifi } from "lucide-react";

import { useTranslation } from "../../i18n";

import "./device-frame.css";

export type DeviceKind = "desktop" | "tablet" | "phone";

export interface DeviceFrameProps {
  /** Physical frame to draw. The viewport size is independent of this. */
  device: DeviceKind;
  /** Logical viewport width in CSS pixels, before scale. */
  width: number;
  /** Logical viewport height in CSS pixels, before scale. */
  height: number;
  scale?: number;
  /** Accessible name for the frame, e.g. "Aperture overview, phone". */
  label: string;
  /** Browser address shown on desktop frames. */
  url?: string;
  selected?: boolean;
  /** Draw the safe-area and thumb-zone instruments over the viewport. */
  guides?: boolean;
  /** Show the device status bar inside the frame. Defaults to true off desktop. */
  statusBar?: boolean;
  onSelect?: () => void;
  children: ReactNode;
}

const DEFAULT_STATUS_BAR: Record<DeviceKind, boolean> = {
  desktop: false,
  tablet: true,
  phone: true,
};

/**
 * A device frame for preview surfaces: browser chrome, iPad, iPhone.
 *
 * The frame owns the chrome (bezel, status bar, home indicator, safe area) and
 * a single zoom. It never re-implements the page: children are the real
 * composition rendered at `width` × `height`, then scaled once.
 */
export function DeviceFrame({
  device,
  width,
  height,
  scale = 1,
  label,
  url = "app.vrooli.local",
  selected = false,
  guides = false,
  statusBar = DEFAULT_STATUS_BAR[device],
  onSelect,
  children,
}: DeviceFrameProps) {
  const { t } = useTranslation();
  const displayWidth = Math.round(width * scale);
  const displayHeight = Math.round(height * scale);
  const safeTop = device === "phone" ? 54 : device === "tablet" ? 24 : 0;

  const style = {
    "--df-scale": String(scale),
    "--df-safe-top": `${safeTop}px`,
  } as CSSProperties;

  const contentStyle: CSSProperties = {
    width,
    height,
    transform: `scale(${scale})`,
    transformOrigin: "top left",
  };

  return (
    <div
      className={`df df--${device}${selected ? " is-selected" : ""}`}
      data-device={device}
      data-testid="device-frame"
      style={style}
    >
      {device === "desktop" ? (
        <div className="df-browser" aria-hidden="true">
          <span className="df-dot" />
          <span className="df-dot" />
          <span className="df-dot" />
          <span className="df-url">{url}</span>
        </div>
      ) : null}

      <div className="df-bezel" style={{ width: displayWidth, height: displayHeight }}>
        <div className="df-viewport" aria-label={label} role="group" style={{ width: displayWidth, height: displayHeight }}>
          <div className="df-content" style={contentStyle}>
            {statusBar ? (
              <div className="df-status" aria-hidden="true">
                <span className="df-time">9:41</span>
                <span className="df-sb">
                  <Signal size={13} strokeWidth={2.2} />
                  <Wifi size={13} strokeWidth={2.2} />
                  <BatteryFull size={15} strokeWidth={2} />
                </span>
              </div>
            ) : null}
            {children}
          </div>

          {device === "phone" ? <span className="df-island" aria-hidden="true" /> : null}
          {device === "tablet" ? <span className="df-camera" aria-hidden="true" /> : null}
          {device !== "desktop" ? <span className="df-home" aria-hidden="true" /> : null}

          {onSelect ? (
            <button
              type="button"
              className="df-select"
              aria-label={t("workbench.studio.selectFrame", {
                defaultValue: "Select {{label}}",
                label,
              })}
              onClick={onSelect}
            />
          ) : null}

          {guides ? (
            <div className="df-guides" aria-hidden="true">
              <span className="df-safe" />
              <span className="df-thumb" />
              <span className="df-guide-label">
                {t("workbench.deviceFrame.thumbZone", { defaultValue: "Thumb zone" })}
              </span>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
