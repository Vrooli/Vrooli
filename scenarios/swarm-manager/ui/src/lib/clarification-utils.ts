import type { ClarificationImpact } from "../types/domain";

/**
 * Parse impact assessment XML from an assistant's clarification response.
 * Returns null if no valid impact block is found (graceful degradation).
 */
export function parseImpactFromContent(content: string): ClarificationImpact | null {
  const blockMatch = content.match(/<impact\s+level="(none|decision|round)">([\s\S]*?)<\/impact>/);
  if (!blockMatch) return null;

  const level = blockMatch[1] as ClarificationImpact["level"];
  const block = blockMatch[2] ?? "";

  const reasoning = block.match(/<reasoning>([\s\S]*?)<\/reasoning>/)?.[1]?.trim() ?? "";
  const context_note = block.match(/<context_note>([\s\S]*?)<\/context_note>/)?.[1]?.trim() ?? "";
  const suggested_update = block.match(/<suggested_update>([\s\S]*?)<\/suggested_update>/)?.[1]?.trim();

  return { level, reasoning, context_note, suggested_update: suggested_update || undefined };
}
