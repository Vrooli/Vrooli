import { useEffect, useRef, useState } from 'react';

export interface ScopedStore<T> { get: () => T; set: (next: T | ((previous: T) => T)) => void; subscribe: (listener: () => void) => () => void; }
export function createScopedStore<T>(initial: T): ScopedStore<T> {
  let value = initial; const listeners = new Set<() => void>();
  return { get: () => value, set: (next) => { value = typeof next === 'function' ? (next as (previous: T) => T)(value) : next; listeners.forEach(listener => listener()); }, subscribe: (listener) => { listeners.add(listener); return () => listeners.delete(listener); } };
}
export interface LayerRecord { id: string; kind: string; dismiss?: () => void; }
const layerRecords: LayerRecord[] = [];
export const layerManager = { push: (layer: LayerRecord) => { layerRecords.push(layer); return () => { const index = layerRecords.indexOf(layer); if (index >= 0) layerRecords.splice(index, 1); }; }, top: () => layerRecords.at(-1), dismissTop: () => layerRecords.at(-1)?.dismiss?.(), list: () => [...layerRecords] };
export interface Command { id: string; label: string; run: () => void; keywords?: string[]; }
const commands = new Map<string, Command>();
export const commandRegistry = { register: (command: Command) => { commands.set(command.id, command); return () => commands.delete(command.id); }, get: (id: string) => commands.get(id), search: (query: string) => [...commands.values()].filter(command => [command.label, ...(command.keywords ?? [])].join(' ').toLowerCase().includes(query.toLowerCase())) };
export interface Shortcut { id: string; keys: string; run: () => void; }
const shortcuts = new Map<string, Shortcut>();
export const shortcutRegistry = { register: (shortcut: Shortcut) => { shortcuts.set(shortcut.id, shortcut); return () => shortcuts.delete(shortcut.id); }, resolve: (keys: string) => [...shortcuts.values()].find(shortcut => shortcut.keys === keys) };
export function useSelectionStore<T>(initial: T[] = []) { const store = useRef(createScopedStore<T[]>(initial)).current; const [selected, setSelected] = useState<T[]>(store.get()); useEffect(() => store.subscribe(() => setSelected(store.get())), [store]); return { selected, setSelected: (next: T[]) => store.set(next), toggle: (item: T) => store.set((current: T[]) => current.includes(item) ? current.filter((value: T) => value !== item) : [...current, item]) }; }
export function useScrollRestoration(key: string) { useEffect(() => { const saved = sessionStorage.getItem(`scroll:${key}`); if (saved) window.scrollTo({ top: Number(saved), behavior: 'instant' as ScrollBehavior }); return () => sessionStorage.setItem(`scroll:${key}`, String(window.scrollY)); }, [key]); }
