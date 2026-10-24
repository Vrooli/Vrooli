import type { DeviceArchetype } from "./deviceArchetype";

const DEVICE_ID_KEY = "wc.deviceId";
const DEVICE_LABEL_KEY = "wc.deviceLabel";

function newID(): string {
  return typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `wc-${Math.random().toString(36).slice(2)}-${Date.now().toString(36)}`;
}

export interface DeviceIdentity {
  id: string;
  label: string;
  /**
   * Display-only device family, used by a follower to frame this client's
   * session. It is derived from `screen`, which — unlike the terminal grid —
   * does not change when a virtual keyboard opens. It is never an
   * authorization signal and never a hardware identity claim.
   */
  deviceClass: DeviceArchetype;
}

export function deviceIdentity(): DeviceIdentity {
  const storedID = localStorage.getItem(DEVICE_ID_KEY);
  const id = storedID || newID();
  if (!storedID) localStorage.setItem(DEVICE_ID_KEY, id);
  const storedLabel = localStorage.getItem(DEVICE_LABEL_KEY);
  const label = storedLabel || defaultDeviceLabel();
  if (!storedLabel) localStorage.setItem(DEVICE_LABEL_KEY, label);
  return { id, label, deviceClass: deviceClassFromScreen() };
}

export function setDeviceLabel(label: string): void { localStorage.setItem(DEVICE_LABEL_KEY, label.trim()); }

/**
 * Classify this device from its physical screen.
 *
 * The label is operator-editable and free-form, so it cannot drive geometry.
 * The terminal grid can, but must not: it shrinks whenever a virtual keyboard
 * opens, which would make a phone reclassify itself as a laptop mid-session.
 * `screen.width`/`screen.height` are stable for the life of the device.
 *
 * The aspect thresholds mirror `archetypeForGrid` so the declared class and
 * the grid-derived fallback agree on the same device.
 */
export function deviceClassFromScreen(): DeviceArchetype {
  if (typeof screen === "undefined") return "laptop";
  const width = screen.width;
  const height = screen.height;
  if (!width || !height) return "laptop";
  const short = Math.min(width, height);
  const long = Math.max(width, height);
  // Touch devices are sized by their short edge; a desktop display is not.
  const touch = typeof navigator !== "undefined" && navigator.maxTouchPoints > 0;
  if (touch && short < 600) return "phone";
  if (touch && short < 1100) return "tablet";
  const aspect = long / short;
  if (aspect >= 2.1) return "ultrawide";
  if (long >= 2000) return "monitor";
  return "laptop";
}

function defaultDeviceLabel(): string {
  const width = typeof screen === "undefined" ? 0 : screen.width;
  const height = typeof screen === "undefined" ? 0 : screen.height;
  const agent = typeof navigator === "undefined" ? "" : navigator.userAgent;
  if (/iPhone/i.test(agent)) return "iPhone";
  if (/iPad|Tablet/i.test(agent)) return "Tablet";
  if (/Android/i.test(agent) && Math.min(width, height) < 900) return "Android phone";
  if (Math.min(width, height) > 0 && Math.min(width, height) < 600) return "Phone";
  if (Math.min(width, height) > 0 && Math.min(width, height) < 900) return "Tablet";
  return "Desktop";
}
