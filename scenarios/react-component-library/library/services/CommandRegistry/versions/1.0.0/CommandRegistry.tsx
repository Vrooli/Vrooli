/** @vrooliComponentSource services.command-registry */

export interface CommandContext {
  query: string;
}

export interface Command {
  id: string;
  label: string;
  description?: string;
  group?: string;
  keywords?: string[];
  shortcut?: string;
  disabled?: boolean;
  run: (context: CommandContext) => void | Promise<void>;
}

export interface CommandRegistrySnapshot {
  commands: readonly Command[];
  recentIds: readonly string[];
}

export interface CommandRegistry {
  register: (command: Command) => () => void;
  unregister: (id: string) => void;
  get: (id: string) => Command | undefined;
  search: (query: string) => Command[];
  execute: (id: string, context?: CommandContext) => Promise<void>;
  getRecent: () => Command[];
  clearRecent: () => void;
  getSnapshot: () => CommandRegistrySnapshot;
  subscribe: (listener: () => void) => () => void;
}

const normalize = (value: string) => value.trim().toLocaleLowerCase();

export function createCommandRegistry(
  initialCommands: Command[] = [],
): CommandRegistry {
  const commands = new Map<string, Command>();
  const recentIds: string[] = [];
  const listeners = new Set<() => void>();
  let snapshot: CommandRegistrySnapshot = {
    commands: [],
    recentIds: [],
  };

  const notify = () => {
    snapshot = {
      commands: [...commands.values()],
      recentIds: [...recentIds],
    };
    listeners.forEach((listener) => listener());
  };

  const remember = (id: string) => {
    const next = [id, ...recentIds.filter((entry) => entry !== id)].slice(0, 8);
    recentIds.splice(0, recentIds.length, ...next);
  };

  const registry: CommandRegistry = {
    register: (command) => {
      if (!command.id.trim() || !command.label.trim()) {
        throw new Error("Commands require a non-empty id and label.");
      }
      commands.set(command.id, command);
      notify();
      return () => {
        if (commands.get(command.id) !== command) return;
        commands.delete(command.id);
        notify();
      };
    },
    unregister: (id) => {
      if (!commands.delete(id)) return;
      notify();
    },
    get: (id) => commands.get(id),
    search: (query) => {
      const needle = normalize(query);
      if (!needle) return [...commands.values()];
      return [...commands.values()].filter((command) => {
        const haystack = normalize(
          [
            command.label,
            command.description ?? "",
            command.group ?? "",
            ...(command.keywords ?? []),
          ].join(" "),
        );
        return haystack.includes(needle);
      });
    },
    execute: async (id, context = { query: "" }) => {
      const command = commands.get(id);
      if (!command || command.disabled) return;
      await command.run(context);
      remember(id);
      notify();
    },
    getRecent: () =>
      recentIds.flatMap((id) => {
        const command = commands.get(id);
        return command ? [command] : [];
      }),
    clearRecent: () => {
      if (!recentIds.length) return;
      recentIds.splice(0, recentIds.length);
      notify();
    },
    getSnapshot: () => snapshot,
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };

  initialCommands.forEach((command) => {
    commands.set(command.id, command);
  });
  notify();
  return registry;
}
