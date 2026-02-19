// DOC: docs/guides/idea-agent-workflow.md
// DOC: docs/guides/idea-agent-workflow.md#implementation-references
import type {
  IdeaClarificationQuestion,
  IdeaSuggestion,
  IdeaSuggestionDecision,
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
    return {
      id: toString(item.id ?? `q${index + 1}`),
      question,
      ...(options ? { options } : {}),
      answer: typeof item.answer === "string" ? item.answer : "",
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

    return {
      id: toString(item.id ?? `s${index + 1}`),
      suggestion,
      details: toString(item.details ?? item.rationale ?? item.context),
      status,
    };
  }
  return null;
};

const withTimestamps = <T extends Record<string, unknown>>(raw: Record<string, unknown> | null, patch: T): T => {
  const now = new Date().toISOString();
  const generatedAt = typeof raw?.generatedAt === "string" ? raw.generatedAt : now;
  return { ...(raw ?? {}), ...patch, generatedAt, updatedAt: now };
};

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
  } catch (error) {
    return {
      raw: null,
      questions: [],
      error: error instanceof Error ? error.message : "Unable to parse questions file.",
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
  } catch (error) {
    return {
      raw: null,
      suggestions: [],
      error: error instanceof Error ? error.message : "Unable to parse suggestions file.",
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
