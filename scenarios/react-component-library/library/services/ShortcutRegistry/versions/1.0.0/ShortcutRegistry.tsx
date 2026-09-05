/** @vrooliComponentSource services.shortcut-registry */

export interface ShortcutContext {
  scope?: string;
  priority?: number;
}

export interface Shortcut {
  id: string;
  keys: string;
  run: () => void | Promise<void>;
  scope?: string;
  priority?: number;
  disabled?: boolean;
}

export interface ShortcutConflict {
  keys: string;
  ids: string[];
}

export interface ShortcutRegistry {
  register: (shortcut: Shortcut) => () => void;
  unregister: (id: string) => void;
  resolve: (keys: string, context?: ShortcutContext) => Shortcut | undefined;
  conflicts: () => ShortcutConflict[];
  subscribe: (listener: () => void) => () => void;
}

const normalize = (keys: string) =>
  keys
    .toLocaleLowerCase()
    .split("+")
    .map((part) => part.trim())
    .map((part) =>
      ["cmd", "command", "control", "ctrl", "meta"].includes(part)
        ? "mod"
        : part,
    )
    .filter(Boolean)
    .sort()
    .join("+");

export function createShortcutRegistry(
  initialShortcuts: Shortcut[] = [],
): ShortcutRegistry {
  const shortcuts = new Map<string, Shortcut>();
  const listeners = new Set<() => void>();
  const notify = () => listeners.forEach((listener) => listener());
  const register = (shortcut: Shortcut) => {
    if (!shortcut.id.trim() || !shortcut.keys.trim()) {
      throw new Error("Shortcuts require a non-empty id and key binding.");
    }
    shortcuts.set(shortcut.id, {
      ...shortcut,
      keys: normalize(shortcut.keys),
    });
    notify();
    return () => {
      if (shortcuts.get(shortcut.id)?.id !== shortcut.id) return;
      shortcuts.delete(shortcut.id);
      notify();
    };
  };

  initialShortcuts.forEach(register);
  return {
    register,
    unregister: (id) => {
      if (!shortcuts.delete(id)) return;
      notify();
    },
    resolve: (keys, context = {}) => {
      const requested = normalize(keys);
      return [...shortcuts.values()]
        .filter((shortcut) => {
          if (shortcut.disabled || normalize(shortcut.keys) !== requested)
            return false;
          if (
            context.scope &&
            shortcut.scope &&
            context.scope !== shortcut.scope
          )
            return false;
          return true;
        })
        .sort((left, right) => (right.priority ?? 0) - (left.priority ?? 0))[0];
    },
    conflicts: () => {
      const grouped = new Map<string, string[]>();
      shortcuts.forEach((shortcut) => {
        const ids = grouped.get(shortcut.keys) ?? [];
        ids.push(shortcut.id);
        grouped.set(shortcut.keys, ids);
      });
      return [...grouped.entries()]
        .filter(([, ids]) => ids.length > 1)
        .map(([keys, ids]) => ({ keys, ids }));
    },
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}

export function eventToShortcut(event: KeyboardEvent) {
  const modifiers = [
    event.metaKey || event.ctrlKey ? "mod" : "",
    event.altKey ? "alt" : "",
    event.shiftKey ? "shift" : "",
  ].filter(Boolean);
  return [...modifiers, event.key.toLocaleLowerCase()].join("+");
}
