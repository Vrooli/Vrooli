import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import type { DeviceArchetype } from "../../lib/deviceArchetype";
import { type ChromeTier, type FollowerRect } from "../../lib/followerViewport";
import { FOLLOWER_ENCLOSURE_Z, FOLLOWER_TRANSITION } from "../../hooks/terminal/useFollowerViewportLayout";
import { strings } from "../../consts/strings";
import { DeviceSilhouette } from "./device/DeviceSilhouette";

export interface DeviceFrameProps {
  archetype: DeviceArchetype;
  chromeTier: ChromeTier;
  rect: FollowerRect;
  /** Share of the screen aperture the leader's keyboard covers, from the bottom. */
  keyboardShare: number;
  /**
   * Distance below the panel where the caption clears the silhouette's stand.
   * It is measured, not a constant: a stand scales with the panel it holds up.
   */
  captionOffset: number;
  leaderDevice: string;
  gridCols: number;
  gridRows: number;
  kbOpen: boolean;
  /**
   * Active terminal palette, so the enclosure belongs to the terminal it
   * surrounds. Only two values cross the boundary; every device token is
   * derived from them in `design-tokens.css`.
   */
  paneTheme: { background?: string; foreground?: string };
  onTakeOver: () => void;
}

// [REQ:P0-002e] The silhouette is decorative chrome around the leader grid. It
// draws a recognisable enclosure for a self-declared device family and never
// asserts which physical hardware the leader is using.
export function DeviceFrame({ archetype, chromeTier, rect, keyboardShare, captionOffset, leaderDevice, gridCols, gridRows, kbOpen, paneTheme, onTakeOver }: DeviceFrameProps) {
  const { t } = useTranslation();
  // Fall back to the translated archetype name rather than leaking the raw
  // enum value into every locale.
  const deviceName = leaderDevice || t(strings.deviceFrame.device[archetype]);
  const caption = t(strings.deviceFrame.caption, { device: deviceName, cols: gridCols, rows: gridRows });
  const description = t(
    kbOpen ? strings.deviceFrame.followingKeyboard : strings.deviceFrame.following,
    { device: deviceName, cols: gridCols, rows: gridRows },
  );

  const strip = chromeTier === "strip";
  const captionClass = strip
    ? "pointer-events-auto absolute left-0 top-0 flex h-7 w-full items-center justify-between gap-2 px-2 text-xs"
    : "pointer-events-auto absolute left-1/2 flex -translate-x-1/2 items-center gap-2 whitespace-nowrap rounded px-2 py-1 text-xs";
  // Monitor-like silhouettes draw a stand below the panel, and it scales with
  // the panel, so the caption clears a measured distance rather than a class.
  const captionStyle = strip ? undefined : { top: `${String(rect.height + captionOffset)}px` };

  const geometry = {
    left: rect.x,
    top: rect.y,
    width: rect.width,
    height: rect.height,
    transition: FOLLOWER_TRANSITION,
    ...themeVariables(paneTheme),
  };

  return <>
    {/* The enclosure is opaque, so it paints *below* the terminal surface.
        Above it, the device would simply hide the session it frames. */}
    <div
      className="pointer-events-none absolute"
      style={{ ...geometry, zIndex: FOLLOWER_ENCLOSURE_Z }}
      data-testid={`device-frame-${chromeTier}`}
      data-archetype={archetype}
      data-keyboard={kbOpen ? "open" : "closed"}
    >
      {strip
        ? <svg aria-hidden="true" className="absolute inset-0 h-full w-full overflow-visible" viewBox="0 0 100 100" preserveAspectRatio="none">
          <path d="M0 1H100" stroke="var(--wc-device-frame-line)" vectorEffect="non-scaling-stroke" />
        </svg>
        : <DeviceSilhouette archetype={archetype} keyboardShare={keyboardShare} kbOpen={kbOpen} />}
    </div>

    {/* The caption and its control are chrome, and stay above everything. */}
    <div className="pointer-events-none absolute z-wc-chrome-raised" style={geometry} data-testid={`device-caption-${chromeTier}`}>
      <div
        className={captionClass}
        style={{
          ...captionStyle,
          background: "var(--wc-device-frame-surface)",
          color: "var(--wc-device-frame-text)",
          border: strip ? undefined : "1px solid var(--wc-device-frame-line)",
          borderRadius: strip ? undefined : "var(--wc-device-frame-caption-radius)",
        }}
      >
        {/* Announce who is driving; the silhouette itself is aria-hidden. */}
        <span role="status" aria-live="polite" className="sr-only">{description}</span>
        <span aria-hidden="true">{caption}</span>
        {kbOpen && !strip && <span
          aria-hidden="true"
          className="rounded-full px-1.5 py-0.5 text-[0.65em] uppercase tracking-wide"
          style={{ color: "var(--wc-device-state-fg)", border: "1px solid var(--wc-device-state-line)" }}
        >{t(strings.deviceFrame.keyboardOpen)}</span>}
        <button
          type="button"
          onClick={onTakeOver}
          className="rounded px-2 py-1 font-medium"
          style={{ background: "var(--wc-device-action-bg)", color: "var(--wc-device-action-fg)" }}
        >{t(strings.deviceFrame.takeOver)}</button>
      </div>
    </div>
  </>;
}

/**
 * Feed the active terminal palette to the device tokens. Absent values are
 * omitted so the CSS fallbacks to the brand palette still apply.
 */
function themeVariables(theme: { background?: string; foreground?: string }): CSSProperties {
  const variables: Record<string, string> = {};
  if (theme.background) {
    variables["--wc-device-theme-bg"] = theme.background;
    variables["--wc-device-theme-surface"] = theme.background;
  }
  if (theme.foreground) variables["--wc-device-theme-fg"] = theme.foreground;
  return variables as CSSProperties;
}
