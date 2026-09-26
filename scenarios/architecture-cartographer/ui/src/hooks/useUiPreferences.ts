import {
  usePersistedPreference,
  type PreferenceStorage,
} from "./usePersistedPreference";

export type Density = "comfortable" | "dense";
export type Handedness = "left" | "right";

export interface UiPreferences {
  density: Density;
  reducedMotion: boolean;
  handedness: Handedness;
  defaultScenario: string;
  defaultDomainFilter: string;
}

const STORAGE_KEY = "cartographer.uiPreferences";

const DEFAULTS: UiPreferences = {
  density: "comfortable",
  reducedMotion: false,
  handedness: "right",
  defaultScenario: "",
  defaultDomainFilter: "",
};

function isDensity(v: unknown): v is Density {
  return v === "comfortable" || v === "dense";
}
function isHandedness(v: unknown): v is Handedness {
  return v === "left" || v === "right";
}

function validate(raw: unknown): UiPreferences | null {
  if (!raw || typeof raw !== "object") return null;
  const r = raw as Record<string, unknown>;
  return {
    density: isDensity(r.density) ? r.density : DEFAULTS.density,
    reducedMotion: typeof r.reducedMotion === "boolean" ? r.reducedMotion : DEFAULTS.reducedMotion,
    handedness: isHandedness(r.handedness) ? r.handedness : DEFAULTS.handedness,
    defaultScenario: typeof r.defaultScenario === "string" ? r.defaultScenario : DEFAULTS.defaultScenario,
    defaultDomainFilter:
      typeof r.defaultDomainFilter === "string" ? r.defaultDomainFilter : DEFAULTS.defaultDomainFilter,
  };
}

export interface UseUiPreferencesOptions {
  storage?: PreferenceStorage;
}

export function useUiPreferences({ storage }: UseUiPreferencesOptions = {}) {
  const [value, setValue] = usePersistedPreference<UiPreferences>({
    key: STORAGE_KEY,
    defaultValue: DEFAULTS,
    validate,
    storage,
  });
  const update = <K extends keyof UiPreferences>(key: K, next: UiPreferences[K]) => {
    setValue({ ...value, [key]: next });
  };
  return { preferences: value, setPreferences: setValue, updatePreference: update };
}

export const UI_PREFERENCE_DEFAULTS: Readonly<UiPreferences> = DEFAULTS;
