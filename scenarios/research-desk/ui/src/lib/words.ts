/**
 * API prose without the command it quotes in backticks: the operator is on a
 * phone and never types a command (rulings V1). The action it names is a
 * button beside it.
 */
export function withoutCommands(text: string): string {
  return text
    .replace(/\s*\(`[^`]*`\)/g, "")
    .replace(/`[^`]*`/g, "")
    .trim();
}
