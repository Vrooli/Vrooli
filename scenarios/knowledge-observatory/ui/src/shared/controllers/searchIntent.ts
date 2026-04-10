// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { SearchMode } from "./searchModes";
import { normalizeSearchMode } from "./searchModes";

export type SearchIntent = {
  mode: SearchMode;
  value: string;
  createdAt: string;
};

const STORAGE_KEY = "ko.search.intent";

const safeSessionStorage = () => {
  if (typeof window === "undefined") return null;
  try {
    return window.sessionStorage;
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to access session storage", error);
    return null;
  }
};

export function storeSearchIntent(intent: Omit<SearchIntent, "createdAt">) {
  const storage = safeSessionStorage();
  if (!storage) return;
  const trimmedValue = intent.value.trim();
  if (!trimmedValue) return;
  const payload: SearchIntent = {
    mode: normalizeSearchMode(intent.mode),
    value: trimmedValue,
    createdAt: new Date().toISOString(),
  };
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to store search intent", error);
  }
}

export function peekSearchIntent(): SearchIntent | null {
  const storage = safeSessionStorage();
  if (!storage) return null;
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as SearchIntent;
    if (!parsed || typeof parsed !== "object") return null;
    return {
      mode: normalizeSearchMode(parsed.mode),
      value: typeof parsed.value === "string" ? parsed.value : "",
      createdAt: typeof parsed.createdAt === "string" ? parsed.createdAt : new Date().toISOString(),
    };
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to parse search intent", error);
    return null;
  }
}

export function consumeSearchIntent(): SearchIntent | null {
  const storage = safeSessionStorage();
  if (!storage) return null;
  const intent = peekSearchIntent();
  try {
    storage.removeItem(STORAGE_KEY);
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to clear search intent", error);
  }
  return intent;
}
