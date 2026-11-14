import { useCallback, useState } from "react";

// Proposal review state that must survive the commit panel unmounting: on
// mobile, opening a file's diff switches panels, and returning must restore
// the open proposal and any unsaved message edits (GCT-053). Session storage
// keeps it per tab; storage failures fall back to in-memory state.

function read<T>(key: string, fallback: T): T {
  try {
    const raw = window.sessionStorage.getItem(key);
    return raw === null ? fallback : (JSON.parse(raw) as T);
  } catch {
    return fallback;
  }
}

export function useSessionState<T>(key: string, fallback: T): [T, (next: T) => void] {
  const [state, setState] = useState<{ key: string; value: T }>(() => ({ key, value: read(key, fallback) }));
  const value = state.key === key ? state.value : read(key, fallback);
  const update = useCallback(
    (next: T) => {
      setState({ key, value: next });
      try {
        if (next === null || next === undefined) {
          window.sessionStorage.removeItem(key);
        } else {
          window.sessionStorage.setItem(key, JSON.stringify(next));
        }
      } catch {
        // In-memory state still works.
      }
    },
    [key],
  );
  return [value, update];
}
