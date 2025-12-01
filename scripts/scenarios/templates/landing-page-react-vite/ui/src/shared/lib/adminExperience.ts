const STORAGE_KEY = 'landing_admin_experience';
const STORAGE_VERSION = 1;

export type AdminExperienceSnapshot = {
  version: number;
  lastVariant?: VariantSessionSnapshot;
  lastAnalytics?: AnalyticsSessionSnapshot;
};

export interface VariantSessionSnapshot {
  slug: string;
  name?: string;
  surface: 'variant' | 'section';
  sectionId?: number;
  sectionType?: string;
  lastVisitedAt: string;
}

export interface AnalyticsSessionSnapshot {
  variantSlug: string | null;
  variantName?: string;
  timeRangeDays: number;
  savedAt: string;
}

const BASE_STATE: AdminExperienceSnapshot = { version: STORAGE_VERSION };

const isBrowser = () => typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';

function readSnapshot(): AdminExperienceSnapshot | null {
  if (!isBrowser()) {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as AdminExperienceSnapshot | null;
    if (!parsed || parsed.version !== STORAGE_VERSION) {
      return null;
    }
    return parsed;
  } catch (error) {
    console.warn('Failed to parse admin experience snapshot:', error);
    return null;
  }
}

function writeSnapshot(snapshot: AdminExperienceSnapshot) {
  if (!isBrowser()) {
    return;
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot));
  } catch (error) {
    console.warn('Failed to persist admin experience snapshot:', error);
  }
}

function updateSnapshot(mutator: (draft: AdminExperienceSnapshot) => void): AdminExperienceSnapshot {
  const current = readSnapshot() ?? { ...BASE_STATE };
  const next: AdminExperienceSnapshot = {
    version: STORAGE_VERSION,
    lastVariant: current.lastVariant,
    lastAnalytics: current.lastAnalytics,
  };
  mutator(next);
  writeSnapshot(next);
  return next;
}

export function getAdminExperienceSnapshot(): AdminExperienceSnapshot {
  return readSnapshot() ?? { ...BASE_STATE };
}

export function rememberVariantSession(params: {
  slug: string;
  name?: string;
  surface: 'variant' | 'section';
  sectionId?: number;
  sectionType?: string;
}) {
  const lastVisitedAt = new Date().toISOString();
  return updateSnapshot((draft) => {
    draft.lastVariant = {
      slug: params.slug,
      name: params.name,
      surface: params.surface,
      sectionId: params.sectionId,
      sectionType: params.sectionType,
      lastVisitedAt,
    };
  });
}

export function rememberAnalyticsFilters(params: {
  variantSlug: string | null;
  variantName?: string;
  timeRangeDays: number;
}) {
  const savedAt = new Date().toISOString();
  return updateSnapshot((draft) => {
    draft.lastAnalytics = {
      variantSlug: params.variantSlug,
      variantName: params.variantName,
      timeRangeDays: params.timeRangeDays,
      savedAt,
    };
  });
}
