/**
 * @libraryId react-component-library:createScopedStore
 * @displayName createScopedStore
 * @description The typed provider and store factory producing isolated store instances per provider, with selector hooks, imperative access, and safe lifecycle teardown.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:createScopedStore
 * @vrooliComponentSourceSlot services.create-scoped-store */

export interface ScopedStore<T> {
  get: () => T;
  set: (next: T | ((previous: T) => T)) => void;
  subscribe: (listener: () => void) => () => void;
}

export function createScopedStore<T>(initial: T): ScopedStore<T> {
  let value = initial;
  const listeners = new Set<() => void>();
  return {
    get: () => value,
    set: (next) => {
      value = typeof next === "function" ? (next as (previous: T) => T)(value) : next;
      listeners.forEach((listener) => listener());
    },
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}
