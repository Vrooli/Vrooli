// DOC: docs/guides/idea-agent-workflow.md
// DOC: docs/guides/idea-agent-workflow.md#implementation-references
import type {
  IdeaClarificationQuestion,
  IdeaSuggestion,
  IdeaSuggestionDecision,
  QuestionLastSynthesis,
  SuggestionLastSynthesis,
  BacklogFile,
} from "../types";

export const IDEA_AGENT_FILE_PATHS = {
  clarify: "clarify/questions.json",
  suggest: "suggest/suggestions.json",
  enhance: "enhance/summary.md",
} as const;

const SUGGESTION_DECISIONS: IdeaSuggestionDecision[] = ["pending", "accepted", "rejected"];

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const toString = (value: unknown): string =>
  typeof value === "string" ? value : value == null ? "" : String(value);

const normalizeOptions = (raw: unknown): string[] | undefined => {
  if (!Array.isArray(raw)) return undefined;
  const filtered = raw
    .map((o: unknown) => (typeof o === "string" ? o : null))
    .filter((o): o is string => o !== null);
  return filtered.length > 0 ? filtered : undefined;
};

const normalizeLastSynthesisQuestion = (raw: unknown): QuestionLastSynthesis | undefined => {
  if (!isRecord(raw)) return undefined;
  if (typeof raw.answer !== "string" || typeof raw.round !== "number") return undefined;
  return { answer: raw.answer, round: raw.round };
};

const normalizeLastSynthesisSuggestion = (raw: unknown): SuggestionLastSynthesis | undefined => {
  if (!isRecord(raw)) return undefined;
  if (typeof raw.status !== "string" || typeof raw.round !== "number") return undefined;
  return { status: raw.status as IdeaSuggestionDecision, round: raw.round };
};

const normalizeQuestion = (item: unknown, index: number): IdeaClarificationQuestion | null => {
  if (typeof item === "string") {
    return { id: `q${index + 1}`, question: item, answer: "" };
  }
  if (isRecord(item)) {
    const question = toString(item.question ?? item.text ?? item.prompt);
    if (!question) {
      return null;
    }
    const options = normalizeOptions(item.options);
    const lastSynthesis = normalizeLastSynthesisQuestion(item.lastSynthesis);
    return {
      id: toString(item.id ?? `q${index + 1}`),
      question,
      ...(options ? { options } : {}),
      answer: typeof item.answer === "string" ? item.answer : "",
      ...(lastSynthesis ? { lastSynthesis } : {}),
    };
  }
  return null;
};

const normalizeSuggestion = (item: unknown, index: number): IdeaSuggestion | null => {
  if (typeof item === "string") {
    return { id: `s${index + 1}`, suggestion: item, status: "pending" };
  }
  if (isRecord(item)) {
    const suggestion = toString(item.suggestion ?? item.title ?? item.text ?? item.recommendation);
    if (!suggestion) {
      return null;
    }
    const statusCandidate = toString(item.status ?? item.decision).toLowerCase();
    const status = SUGGESTION_DECISIONS.includes(statusCandidate as IdeaSuggestionDecision)
      ? (statusCandidate as IdeaSuggestionDecision)
      : "pending";
    const lastSynthesis = normalizeLastSynthesisSuggestion(item.lastSynthesis);

    const notes = toString(item.notes ?? item.note ?? item.response);
    return {
      id: toString(item.id ?? `s${index + 1}`),
      suggestion,
      details: toString(item.details ?? item.rationale ?? item.context),
      status,
      ...(notes ? { notes } : {}),
      ...(lastSynthesis ? { lastSynthesis } : {}),
    };
  }
  return null;
};

const withTimestamps = <T extends Record<string, unknown>>(raw: Record<string, unknown> | null, patch: T): T => {
  const now = new Date().toISOString();
  const generatedAt = typeof raw?.generatedAt === "string" ? raw.generatedAt : now;
  return { ...(raw ?? {}), ...patch, generatedAt, updatedAt: now };
};

/**
 * Attempt to repair truncated JSON by finding the last complete array element
 * within a named array field (e.g. "questions" or "suggestions").
 * Works by tracking brace depth to locate the last fully-closed object,
 * then truncates and re-closes the JSON structure.
 * Returns null if no repair is possible.
 */
function repairTruncatedJson(
  content: string,
  arrayKey: string,
): { parsed: unknown; warning: string } | null {
  const keyIdx = content.indexOf(`"${arrayKey}"`);
  if (keyIdx === -1) return null;
  const arrayStart = content.indexOf("[", keyIdx);
  if (arrayStart === -1) return null;

  // Walk forward through the array, tracking the last position where a
  // top-level object (depth 0 → 1 → 0) closes with "}".
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

export function parseClarifyQuestionsFile(content?: string | null): {
  raw: Record<string, unknown> | null;
  questions: IdeaClarificationQuestion[];
  error?: string;
} {
  if (!content) {
    return { raw: null, questions: [] };
  }

  try {
    const parsed: unknown = JSON.parse(content);
    if (!isRecord(parsed)) {
      return { raw: null, questions: [], error: "Questions file is not a JSON object." };
    }

    const rawQuestions = Array.isArray(parsed.questions) ? parsed.questions : [];
    const questions = rawQuestions
      .map((item, index) => normalizeQuestion(item, index))
      .filter((item): item is IdeaClarificationQuestion => Boolean(item));

    return { raw: parsed, questions };
  } catch (parseError) {
    // Attempt to salvage complete questions from truncated JSON
    const repaired = repairTruncatedJson(content, "questions");
    if (repaired && isRecord(repaired.parsed)) {
      const rawQuestions = Array.isArray(repaired.parsed.questions) ? repaired.parsed.questions : [];
      const questions = rawQuestions
        .map((item, index) => normalizeQuestion(item, index))
        .filter((item): item is IdeaClarificationQuestion => Boolean(item));
      if (questions.length > 0) {
        return { raw: repaired.parsed as Record<string, unknown>, questions, error: repaired.warning };
      }
    }
    return {
      raw: null,
      questions: [],
      error: `Unable to parse clarify/questions.json: ${parseError instanceof Error ? parseError.message : "Unknown error"}`,
    };
  }
}

export function buildClarifyQuestionsContent(
  raw: Record<string, unknown> | null,
  questions: IdeaClarificationQuestion[]
): string {
  const payload = withTimestamps(raw, { questions });
  return JSON.stringify(payload, null, 2);
}

export function parseSuggestionsFile(content?: string | null): {
  raw: Record<string, unknown> | null;
  suggestions: IdeaSuggestion[];
  error?: string;
} {
  if (!content) {
    return { raw: null, suggestions: [] };
  }

  try {
    const parsed: unknown = JSON.parse(content);
    if (!isRecord(parsed)) {
      return { raw: null, suggestions: [], error: "Suggestions file is not a JSON object." };
    }

    const rawSuggestions = Array.isArray(parsed.suggestions) ? parsed.suggestions : [];
    const suggestions = rawSuggestions
      .map((item, index) => normalizeSuggestion(item, index))
      .filter((item): item is IdeaSuggestion => Boolean(item));

    return { raw: parsed, suggestions };
  } catch (parseError) {
    // Attempt to salvage complete suggestions from truncated JSON
    const repaired = repairTruncatedJson(content, "suggestions");
    if (repaired && isRecord(repaired.parsed)) {
      const rawSuggestions = Array.isArray(repaired.parsed.suggestions) ? repaired.parsed.suggestions : [];
      const suggestions = rawSuggestions
        .map((item, index) => normalizeSuggestion(item, index))
        .filter((item): item is IdeaSuggestion => Boolean(item));
      if (suggestions.length > 0) {
        return { raw: repaired.parsed as Record<string, unknown>, suggestions, error: repaired.warning };
      }
    }
    return {
      raw: null,
      suggestions: [],
      error: `Unable to parse suggest/suggestions.json: ${parseError instanceof Error ? parseError.message : "Unknown error"}`,
    };
  }
}

export function buildSuggestionsContent(
  raw: Record<string, unknown> | null,
  suggestions: IdeaSuggestion[]
): string {
  const payload = withTimestamps(raw, { suggestions });
  return JSON.stringify(payload, null, 2);
}

/**
 * Parse questions from a pre-parsed object (as returned inline from the summary endpoint).
 * Skips JSON.parse since the data arrives already deserialized.
 */
export function parseClarifyQuestionsObject(content: Record<string, unknown> | null): {
  raw: Record<string, unknown> | null;
  questions: IdeaClarificationQuestion[];
} {
  if (!content) {
    return { raw: null, questions: [] };
  }
  const rawQuestions = Array.isArray(content.questions) ? content.questions : [];
  const questions = rawQuestions
    .map((item, index) => normalizeQuestion(item, index))
    .filter((item): item is IdeaClarificationQuestion => Boolean(item));
  return { raw: content, questions };
}

/**
 * Parse suggestions from a pre-parsed object (as returned inline from the summary endpoint).
 * Skips JSON.parse since the data arrives already deserialized.
 */
export function parseSuggestionsObject(content: Record<string, unknown> | null): {
  raw: Record<string, unknown> | null;
  suggestions: IdeaSuggestion[];
} {
  if (!content) {
    return { raw: null, suggestions: [] };
  }
  const rawSuggestions = Array.isArray(content.suggestions) ? content.suggestions : [];
  const suggestions = rawSuggestions
    .map((item, index) => normalizeSuggestion(item, index))
    .filter((item): item is IdeaSuggestion => Boolean(item));
  return { raw: content, suggestions };
}

// ---------------------------------------------------------------------------
// Synthesis status helpers — used by UI panels to show per-item indicators.
// ---------------------------------------------------------------------------

export type SynthesisStatus = "new" | "updated" | "incorporated";

export function getQuestionSynthesisStatus(q: IdeaClarificationQuestion): SynthesisStatus {
  if (!q.lastSynthesis) return "new";
  if (q.lastSynthesis.answer !== (q.answer ?? "")) return "updated";
  return "incorporated";
}

export function getSuggestionSynthesisStatus(s: IdeaSuggestion): SynthesisStatus {
  if (!s.lastSynthesis) return "new";
  if (s.lastSynthesis.status !== (s.status ?? "pending")) return "updated";
  return "incorporated";
}

export interface SynthesisSummary {
  incorporated: number;
  updated: number;
  new: number;
}

export function computeQuestionsSynthesisSummary(questions: IdeaClarificationQuestion[]): SynthesisSummary {
  const summary: SynthesisSummary = { incorporated: 0, updated: 0, new: 0 };
  for (const q of questions) {
    summary[getQuestionSynthesisStatus(q)]++;
  }
  return summary;
}

export function computeSuggestionsSynthesisSummary(suggestions: IdeaSuggestion[]): SynthesisSummary {
  const summary: SynthesisSummary = { incorporated: 0, updated: 0, new: 0 };
  for (const s of suggestions) {
    summary[getSuggestionSynthesisStatus(s)]++;
  }
  return summary;
}

export function findBacklogFileByPath(files: BacklogFile[] | undefined, targetPath: string): BacklogFile | null {
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
