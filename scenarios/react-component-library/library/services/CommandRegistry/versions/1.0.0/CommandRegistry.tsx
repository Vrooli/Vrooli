/** @vrooliComponentSource services.command-registry */

export interface Command {
  id: string;
  label: string;
  run: () => void;
  keywords?: string[];
}

const commands = new Map<string, Command>();

export const commandRegistry = {
  register: (command: Command) => {
    commands.set(command.id, command);
    return () => commands.delete(command.id);
  },
  get: (id: string) => commands.get(id),
  search: (query: string) =>
    [...commands.values()].filter((command) =>
      [command.label, ...(command.keywords ?? [])]
        .join(" ")
        .toLowerCase()
        .includes(query.toLowerCase()),
    ),
};
