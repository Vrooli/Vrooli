/**
 * Workshop file parsing utilities.
 *
 * Handles reading and writing workshop round JSON files with
 * truncation recovery for robustness against agent crashes mid-write.
 *
 * DOC: docs/internal/SEAMS.md#workshop-parsing
 */

import type { BacklogFile, WorkshopItem, WorkshopRound } from "../types/domain";

export const WORKSHOP_FILE_PATHS = {
  plan: "plan.md",
  workshopDir: "workshop/",
} as const;

// ---------------------------------------------------------------------------
// Truncation recovery (ported from idea-agent-files.ts)
// ---------------------------------------------------------------------------

function repairTruncatedJson(
  content: string,
  arrayKey: string,
): { parsed: unknown; warning: string } | null {
  const keyIdx = content.indexOf(`"${arrayKey}"`);
  if (keyIdx === -1) return null;
  const arrayStart = content.indexOf("[", keyIdx);
  if (arrayStart === -1) return null;

  let lastGoodEnd = -1;
  let depth = 0;
  let inString = false;
  let escaped = false;

  for (let i = arrayStart + 1; i < content.length; i++) {
    const ch = content[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (ch === "\\") {
      escaped = true;
      continue;
    }
    if (ch === '"') {
      inString = !inString;
      continue;
    }
    if (inString) continue;

    if (ch === "{" || ch === "[") {
      depth++;
    } else if (ch === "}" || ch === "]") {
      depth--;
      if (depth === 0 && ch === "}") {
        lastGoodEnd = i;
      }
    }
  }

  if (lastGoodEnd === -1) return null;

  const repaired = content.slice(0, lastGoodEnd + 1) + "\n  ]\n}";
  try {
    const parsed = JSON.parse(repaired) as Record<string, unknown>;
    const totalInFile = (content.match(/"id"\s*:/g) || []).length;
    const arr = parsed[arrayKey];
    const recovered = Array.isArray(arr) ? arr.length : 0;
    const warning = `File appears truncated. Recovered ${recovered} of ~${totalInFile} item(s).`;
    return { parsed, warning };
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Workshop round parsing
// ---------------------------------------------------------------------------

export function parseWorkshopRound(content?: string | null): {
  round: WorkshopRound | null;
  error?: string;
} {
  if (!content || !content.trim()) {
    return { round: null };
  }

  try {
    const parsed = JSON.parse(content) as WorkshopRound;
    if (!parsed || typeof parsed !== "object") {
      return { round: null, error: "Invalid workshop round format" };
    }
    // Normalize items
    if (Array.isArray(parsed.items)) {
      parsed.items = (parsed.items as unknown[]).map((i) => normalizeWorkshopItem(i as Record<string, unknown>));
    } else {
      parsed.items = [];
    }
    return { round: parsed };
  } catch {
    // Attempt truncation recovery
    const repaired = repairTruncatedJson(content, "items");
    if (repaired) {
      const round = repaired.parsed as WorkshopRound;
      if (Array.isArray(round.items)) {
        round.items = (round.items as unknown[]).map((i) => normalizeWorkshopItem(i as Record<string, unknown>));
      }
      return { round, error: repaired.warning };
    }
    return { round: null, error: "Failed to parse workshop round JSON" };
  }
}

function normalizeDecisionOption(raw: unknown): import("../types/domain").DecisionOption {
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    const obj = raw as Record<string, unknown>;
    return {
      key: String(obj.key ?? ""),
      label: String(obj.label ?? ""),
      rationale: String(obj.rationale ?? ""),
    };
  }
  // Fallback for legacy string options
  const str = String(raw ?? "");
  return { key: str, label: str, rationale: "" };
}

function normalizeWorkshopItem(raw: Record<string, unknown>): WorkshopItem {
  const type = String(raw.type ?? "info");
  const item: WorkshopItem = {
    id: String(raw.id ?? ""),
    type: type as WorkshopItem["type"],
  };

  if (raw.topic) item.topic = String(raw.topic);
  if (raw.text) item.text = String(raw.text);
  if (raw.context) item.context = String(raw.context);

  if (Array.isArray(raw.options)) {
    item.options = raw.options.map(normalizeDecisionOption);
  }

  if (type === "decision") {
    item.selected = raw.selected === null || raw.selected === undefined
      ? null
      : String(raw.selected);
    item.freeform = raw.freeform === null || raw.freeform === undefined
      ? null
      : String(raw.freeform);
    item.notes = raw.notes === null || raw.notes === undefined
      ? null
      : String(raw.notes);
  }

  return item;
}

// ---------------------------------------------------------------------------
// Workshop round building
// ---------------------------------------------------------------------------

export function buildWorkshopRoundContent(round: WorkshopRound): string {
  return JSON.stringify(round, null, 2);
}

// ---------------------------------------------------------------------------
// Workshop round metrics
// ---------------------------------------------------------------------------

export function getPendingDecisionCount(round: WorkshopRound): number {
  return round.items.filter(
    (item) => item.type === "decision" && (!item.selected || !item.selected.trim()),
  ).length;
}

// ---------------------------------------------------------------------------
// File tree utilities (shared across backlog views)
// ---------------------------------------------------------------------------

export function findBacklogFileByPath(
  files: BacklogFile[] | undefined,
  targetPath: string,
): BacklogFile | null {
  if (!files || files.length === 0) return null;
  for (const file of files) {
    if (file.path === targetPath) {
      return file;
    }
    if (file.children && file.children.length > 0) {
      const match = findBacklogFileByPath(file.children, targetPath);
      if (match) {
        return match;
      }
    }
  }
  return null;
}
