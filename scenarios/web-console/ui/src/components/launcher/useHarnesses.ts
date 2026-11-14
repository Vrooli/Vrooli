import { useEffect, useState } from "react";
import { capabilitiesClient } from "../../api/capabilities";

export interface Harness {
  agentId: string;
  displayName: string;
  /** Windows this harness says it will report. They only reserve card space. */
  declaredWindows: number;
  supportsContextLimit: boolean;
  supportsUsageMode: boolean;
  supportsModelSelection: boolean;
  supportsProviderSelection: boolean;
}

/** Rows reserved before any target has answered, so a first open rarely shifts. */
const FIRST_OPEN_SLOTS = 2;

/** The last answer per target, so reopening the drawer never shifts cards. */
const lastKnown = new Map<string, Record<string, Harness>>();

export interface HarnessCatalog {
  harnesses: Record<string, Harness>;
  /** False while loading or when the server cannot answer: features are unknown, not absent. */
  known: boolean;
}

/** Harness identity and features for a target, read from each resource's descriptor. */
export function useHarnesses(enabled: boolean, targetId: string): HarnessCatalog {
  const [state, setState] = useState<HarnessCatalog>(() => ({ harnesses: lastKnown.get(targetId) ?? {}, known: lastKnown.has(targetId) }));
  useEffect(() => {
    setState({ harnesses: lastKnown.get(targetId) ?? {}, known: lastKnown.has(targetId) });
    if (!enabled) return;
    const abort = new AbortController();
    capabilitiesClient.listHarnesses({ targetId }, { signal: abort.signal }).then((response) => {
      if (abort.signal.aborted) return;
      const harnesses = Object.fromEntries(response.harnesses.map((item) => [item.agentId, {
        agentId: item.agentId,
        displayName: item.displayName,
        declaredWindows: item.usageWindows.length,
        supportsContextLimit: item.supportsContextLimit,
        supportsUsageMode: item.supportsUsageMode,
        supportsModelSelection: item.supportsModelSelection,
        supportsProviderSelection: item.supportsProviderSelection,
      } satisfies Harness]));
      lastKnown.set(targetId, harnesses);
      setState({ harnesses, known: true });
    }).catch(() => {
      // An older server. The launcher stays usable; descriptor-owned rows degrade.
    });
    return () => { abort.abort(); };
  }, [enabled, targetId]);
  return state;
}

/**
 * Windows to reserve for one harness. While descriptors are unknown the card
 * still holds a stable height and shows placeholders rather than guessing that
 * a harness reports nothing.
 */
export function declaredWindowsFor(catalog: HarnessCatalog, agentId: string): number {
  if (catalog.known) return catalog.harnesses[agentId]?.declaredWindows ?? 0;
  return FIRST_OPEN_SLOTS;
}

/** Forget cached descriptors. Test isolation only. */
export function resetHarnessCacheForTests() {
  lastKnown.clear();
}
