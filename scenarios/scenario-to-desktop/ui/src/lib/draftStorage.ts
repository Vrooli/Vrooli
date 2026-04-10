/**
 * Local storage for generator app state (UI preferences only).
 * Form state and pipeline results are now stored server-side via useScenarioState.
 */

export type GeneratorAppState = {
  version: number;
  updatedAt: string;
  viewMode: string;
  selectedScenarioName: string;
  selectedTemplate: string;
  selectionSource: "inventory" | "manual" | null;
  currentBuildId: string | null;
  installerBuildId?: string | null;
  activeStep: number;
  userPinnedStep: boolean;
  docPath: string | null;
};

const APP_STATE_KEY = "std_generator_app_state_v2";
const APP_STATE_VERSION = 2;

function nowISO(): string {
  return new Date().toISOString();
}

function getWindowStorage(): Storage | null {
  if (typeof window === "undefined") return null;
  return window.localStorage;
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

export function saveGeneratorAppState(state: Partial<GeneratorAppState>) {
  const storage = getWindowStorage();
  if (!storage) return;
  const existing = loadGeneratorAppState() || {
    version: APP_STATE_VERSION,
    updatedAt: nowISO(),
    viewMode: "wizard",
    selectedScenarioName: "",
    selectedTemplate: "basic",
    selectionSource: null,
    currentBuildId: null,
    installerBuildId: null,
    activeStep: 0,
    userPinnedStep: false,
    docPath: null,
  };
  const payload: GeneratorAppState = {
    ...existing,
    ...state,
    version: APP_STATE_VERSION,
    updatedAt: nowISO(),
  };
  storage.setItem(APP_STATE_KEY, JSON.stringify(payload));
}

export function clearGeneratorAppState() {
  const storage = getWindowStorage();
  if (!storage) return;
  storage.removeItem(APP_STATE_KEY);
}
