/** Two-letter initials from a display name or address, never empty. */
export function initials(value: string | undefined | null): string {
  const text = (value ?? "").trim();
  if (!text) return "?";
  const words = text.replace(/^[@+]/, "").split(/[\s._-]+/).filter(Boolean);
  if (words.length === 0) return text.slice(0, 2).toUpperCase();
  if (words.length === 1) return (words[0] ?? "").slice(0, 2).toUpperCase();
  return `${(words[0] ?? "").charAt(0)}${(words[1] ?? "").charAt(0)}`.toUpperCase();
}

/** Deterministic hue (0–359) from any string, for identity colouring. */
export function hueFor(value: string): number {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) >>> 0;
  }
  return hash % 360;
}

/** The thread's human-facing title: a display name, else the address/key. */
export function threadTitle(input: { display_name?: string; sender_address?: string; thread_key?: string }): string {
  return input.display_name?.trim() || input.sender_address?.trim() || input.thread_key?.trim() || "";
}

export function truncate(value: string, max: number): string {
  const text = value.replace(/\s+/g, " ").trim();
  return text.length > max ? `${text.slice(0, Math.max(0, max - 1)).trimEnd()}…` : text;
}
