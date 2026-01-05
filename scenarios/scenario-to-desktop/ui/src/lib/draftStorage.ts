type DraftRecord<T> = {
  scenarioName: string;
  version: number;
  createdAt: string;
  updatedAt: string;
  expiresAt: string;
  data: T;
};

type GeneratorAppState = {
  version: number;
  updatedAt: string;
  viewMode: string;
  selectedScenarioName: string;
  selectedTemplate: string;
  selectionSource: "inventory" | "manual" | null;
  currentBuildId: string | null;
  activeStep: number;
  userPinnedStep: boolean;
  docPath: string | null;
};

const DRAFT_STORAGE_KEY = "std_generator_drafts_v1";
const APP_STATE_KEY = "std_generator_app_state_v1";
const DRAFT_VERSION = 1;
const APP_STATE_VERSION = 1;
const MAX_DRAFTS = 6;
const DRAFT_TTL_MS = 1000 * 60 * 60 * 24 * 7;

function nowISO(): string {
  return new Date().toISOString();
}

function getWindowStorage(): Storage | null {
  if (typeof window === "undefined") return null;
  return window.localStorage;
}

function parseDrafts(raw: string | null): DraftRecord<unknown>[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as DraftRecord<unknown>[];
  } catch {
    return [];
  }
}

function readDrafts(): DraftRecord<unknown>[] {
  const storage = getWindowStorage();
  if (!storage) return [];
  return parseDrafts(storage.getItem(DRAFT_STORAGE_KEY));
}

function writeDrafts(drafts: DraftRecord<unknown>[]) {
  const storage = getWindowStorage();
  if (!storage) return;
  storage.setItem(DRAFT_STORAGE_KEY, JSON.stringify(drafts));
}

function isExpired(record: DraftRecord<unknown>): boolean {
  if (!record.expiresAt) return false;
  return Date.parse(record.expiresAt) <= Date.now();
}

function cleanupDrafts(drafts: DraftRecord<unknown>[]): DraftRecord<unknown>[] {
  return drafts.filter((draft) => !isExpired(draft));
}

export function loadGeneratorDraft<T>(scenarioName: string): DraftRecord<T> | null {
  const drafts = cleanupDrafts(readDrafts());
  const match = drafts.find((draft) => draft.scenarioName === scenarioName) as DraftRecord<T> | undefined;
  if (drafts.length > 0) {
    writeDrafts(drafts);
  }
  return match || null;
}

export function saveGeneratorDraft<T>(scenarioName: string, data: T): DraftRecord<T> {
  const drafts = cleanupDrafts(readDrafts());
  const now = nowISO();
  const existingIndex = drafts.findIndex((draft) => draft.scenarioName === scenarioName);
  const createdAt = existingIndex >= 0 ? drafts[existingIndex].createdAt : now;
  const record: DraftRecord<T> = {
    scenarioName,
    version: DRAFT_VERSION,
    createdAt,
    updatedAt: now,
    expiresAt: new Date(Date.now() + DRAFT_TTL_MS).toISOString(),
    data
  };

  if (existingIndex >= 0) {
    drafts[existingIndex] = record as DraftRecord<unknown>;
  } else {
    drafts.push(record as DraftRecord<unknown>);
  }

  drafts.sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
  const trimmed = drafts.slice(0, MAX_DRAFTS);
  writeDrafts(trimmed);
  return record;
}

export function clearGeneratorDraft(scenarioName: string) {
  const drafts = cleanupDrafts(readDrafts()).filter((draft) => draft.scenarioName !== scenarioName);
  writeDrafts(drafts);
}

export function clearAllGeneratorDrafts() {
  writeDrafts([]);
}

export function loadGeneratorAppState(): GeneratorAppState | null {
  const storage = getWindowStorage();
  if (!storage) return null;
  try {
    const raw = storage.getItem(APP_STATE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as GeneratorAppState;
    if (!parsed || parsed.version !== APP_STATE_VERSION) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function saveGeneratorAppState(state: GeneratorAppState) {
  const storage = getWindowStorage();
  if (!storage) return;
  const payload = {
    ...state,
    version: APP_STATE_VERSION,
    updatedAt: nowISO()
  };
  storage.setItem(APP_STATE_KEY, JSON.stringify(payload));
}

export function clearGeneratorAppState() {
  const storage = getWindowStorage();
  if (!storage) return;
  storage.removeItem(APP_STATE_KEY);
}

export type { DraftRecord, GeneratorAppState };
