export type ActivityStatus = "running" | "completed" | "failed" | "info";

export type ActivityRecord = {
  id: string;
  type: string;
  title: string;
  description?: string;
  status?: ActivityStatus;
  createdAt: string;
  meta?: Record<string, string>;
};

const STORAGE_KEY = "ko.activity.feed";
const MAX_ITEMS = 40;
const UPDATE_EVENT = "ko-activity-updated";

const safeLocalStorage = () => {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage;
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to access local storage", error);
    return null;
  }
};

const emitUpdate = () => {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT));
};

const buildId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `activity-${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

export function loadActivityFeed(): ActivityRecord[] {
  const storage = safeLocalStorage();
  if (!storage) return [];
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as ActivityRecord[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => item && typeof item === "object" && typeof item.id === "string");
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to parse activity feed", error);
    return [];
  }
}

export function recordActivity(entry: Omit<ActivityRecord, "id" | "createdAt"> & { createdAt?: string }) {
  const storage = safeLocalStorage();
  if (!storage) return;
  const trimmedTitle = entry.title.trim();
  if (!trimmedTitle) return;
  const record: ActivityRecord = {
    id: buildId(),
    type: entry.type,
    title: trimmedTitle,
    description: entry.description?.trim() || undefined,
    status: entry.status,
    createdAt: entry.createdAt ?? new Date().toISOString(),
    meta: entry.meta,
  };
  const existing = loadActivityFeed();
  const next = [record, ...existing].slice(0, MAX_ITEMS);
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to persist activity feed", error);
  }
  emitUpdate();
}

export function subscribeActivityUpdates(listener: () => void) {
  if (typeof window === "undefined") return () => undefined;
  window.addEventListener(UPDATE_EVENT, listener);
  return () => window.removeEventListener(UPDATE_EVENT, listener);
}

export function clearActivityFeed() {
  const storage = safeLocalStorage();
  if (!storage) return;
  try {
    storage.removeItem(STORAGE_KEY);
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to clear activity feed", error);
  }
  emitUpdate();
}
