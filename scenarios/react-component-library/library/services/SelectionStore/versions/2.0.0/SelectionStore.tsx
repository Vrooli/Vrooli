/** @libraryId react-component-library:SelectionStore */
/** @vrooliComponentSource services.selection-store */
import { useMemo, useRef, useSyncExternalStore } from "react";
export type SelectionMode = "none" | "single" | "multi";
export interface SelectionSnapshot { keys: ReadonlySet<string>; anchorKey: string | null; mode: SelectionMode; }
export interface SelectionStore { getSnapshot(): SelectionSnapshot; subscribe(listener: () => void): () => void; setMode(mode: SelectionMode): void; setSelected(keys: readonly string[]): void; toggle(key: string): void; select(key: string): void; extendTo(key: string, ordered: readonly string[]): void; selectAll(ordered: readonly string[]): void; invert(ordered: readonly string[]): void; clear(): void; retain(visible: readonly string[], policy: "prune" | "keep"): void; isSelected(key: string): boolean; size(): number; }
export function createSelectionStore(initial: readonly string[] = [], mode: SelectionMode = "multi"): SelectionStore {
  let snapshot: SelectionSnapshot = { keys: new Set(initial), anchorKey: initial.at(-1) ?? null, mode };
  const listeners = new Set<() => void>();
  const publish = (next: SelectionSnapshot) => { snapshot = next; listeners.forEach((listener) => listener()); };
  const mutate = (keys: Iterable<string>, anchorKey = snapshot.anchorKey, nextMode = snapshot.mode) => publish({ keys: new Set(keys), anchorKey, mode: nextMode });
  return { getSnapshot: () => snapshot, subscribe: (listener) => { listeners.add(listener); return () => listeners.delete(listener); }, setMode: (next) => next === "none" ? mutate([], null, next) : publish({ ...snapshot, mode: next }), setSelected: (keys) => mutate(snapshot.mode === "single" ? [...keys].slice(0, 1) : keys, [...keys].at(-1) ?? null), toggle: (key) => { if (snapshot.mode === "none") return; const keys = new Set(snapshot.keys); if (keys.has(key)) keys.delete(key); else { if (snapshot.mode === "single") keys.clear(); keys.add(key); } mutate(keys, key); }, select: (key) => snapshot.mode !== "none" && mutate([key], key), extendTo: (key, ordered) => { if (snapshot.mode !== "multi") return; const anchor = snapshot.anchorKey; if (!anchor) return mutate([key], key); const from = ordered.indexOf(anchor), to = ordered.indexOf(key); if (from < 0 || to < 0) return mutate([key], anchor); const [start, end] = from <= to ? [from, to] : [to, from]; mutate(ordered.slice(start, end + 1), anchor); }, selectAll: (ordered) => snapshot.mode !== "none" && mutate(snapshot.mode === "single" ? ordered.slice(0, 1) : ordered), invert: (ordered) => { const current = new Set(snapshot.keys); mutate(ordered.filter((key) => !current.has(key)), snapshot.anchorKey); }, clear: () => mutate([], null), retain: (visible, policy) => policy === "keep" ? publish({ ...snapshot }) : mutate([...snapshot.keys].filter((key) => visible.includes(key)), snapshot.anchorKey), isSelected: (key) => snapshot.keys.has(key), size: () => snapshot.keys.size };
}
export function useSelectionStore<T extends string>(initial: T[] = []) {
  const ref = useRef<SelectionStore>();
  if (!ref.current) ref.current = createSelectionStore(initial);
  const store = ref.current;
  const snapshot = useSyncExternalStore(
    (listener) => store.subscribe(listener),
    () => store.getSnapshot(),
    () => store.getSnapshot(),
  );
  return useMemo(() => ({
    getSnapshot: () => store.getSnapshot(),
    subscribe: (listener: () => void) => store.subscribe(listener),
    setMode: (mode: SelectionMode) => store.setMode(mode),
    setSelected: (next: T[]) => { store.clear(); next.forEach((key) => store.toggle(key)); },
    toggle: (key: T) => store.toggle(key),
    select: (key: T) => store.select(key),
    extendTo: (key: T, ordered: readonly string[]) => store.extendTo(key, ordered),
    selectAll: (ordered: readonly string[]) => store.selectAll(ordered),
    invert: (ordered: readonly string[]) => store.invert(ordered),
    clear: () => store.clear(),
    retain: (visible: readonly string[], policy: "prune" | "keep") => store.retain(visible, policy),
    isSelected: (key: T) => store.isSelected(key),
    size: () => store.size(),
    selected: [...snapshot.keys] as T[],
  }), [snapshot, store]);
}
