const DEVICE_ID_KEY = "wc.deviceId";
const DEVICE_LABEL_KEY = "wc.deviceLabel";

function newID(): string {
  return typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `wc-${Math.random().toString(36).slice(2)}-${Date.now().toString(36)}`;
}

export function deviceIdentity(): { id: string; label: string } {
  const storedID = localStorage.getItem(DEVICE_ID_KEY);
  const id = storedID || newID();
  if (!storedID) localStorage.setItem(DEVICE_ID_KEY, id);
  const storedLabel = localStorage.getItem(DEVICE_LABEL_KEY);
  const label = storedLabel || defaultDeviceLabel();
  if (!storedLabel) localStorage.setItem(DEVICE_LABEL_KEY, label);
  return { id, label };
}

export function setDeviceLabel(label: string): void { localStorage.setItem(DEVICE_LABEL_KEY, label.trim()); }

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
