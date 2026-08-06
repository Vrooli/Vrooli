/** @vrooliComponentSource services.shortcut-registry */

export interface Shortcut {
  id: string;
  keys: string;
  run: () => void;
}

const shortcuts = new Map<string, Shortcut>();

export const shortcutRegistry = {
  register: (shortcut: Shortcut) => {
    shortcuts.set(shortcut.id, shortcut);
    return () => shortcuts.delete(shortcut.id);
  },
  resolve: (keys: string) =>
    [...shortcuts.values()].find((shortcut) => shortcut.keys === keys),
};
