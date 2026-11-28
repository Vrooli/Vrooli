/** Shared same-origin storage boundary for local owner-session state. */
export class BrowserStorageUnavailableError extends Error {
  constructor() {
    super("Browser storage is unavailable");
    this.name = "BrowserStorageUnavailableError";
  }
}

function storage(): Storage | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage;
  } catch {
    throw new BrowserStorageUnavailableError();
  }
}

export function readBrowserStorage(key: string): string | null {
  const target = storage();
  if (!target) return null;
  try {
    return target.getItem(key);
  } catch {
    throw new BrowserStorageUnavailableError();
  }
}

export function writeBrowserStorage(key: string, value: string): void {
  const target = storage();
  if (!target) throw new BrowserStorageUnavailableError();
  try {
    target.setItem(key, value);
  } catch {
    throw new BrowserStorageUnavailableError();
  }
}

export function removeBrowserStorage(key: string): void {
  const target = storage();
  if (!target) return;
  try {
    target.removeItem(key);
  } catch {
    throw new BrowserStorageUnavailableError();
  }
}
