// One translation from a transport failure to what an operator reads: the
// cause in a sentence, what to do about it, whether retrying can help, and the
// verbatim detail kept for engineers. Every surface renders errors through this
// so a raw "[unauthenticated] HTTP 401" never becomes body copy.

export type ErrorKind = "auth" | "forbidden" | "unreachable" | "misrouted" | "not_found" | "timeout" | "unimplemented" | "rejected" | "unknown";

export interface ErrorPresentation {
  kind: ErrorKind;
  title: string;
  message: string;
  /** The original message, verbatim; shown collapsed. */
  detail: string;
  retryable: boolean;
}

const COPY: Record<Exclude<ErrorKind, "rejected" | "unknown">, Omit<ErrorPresentation, "kind" | "detail">> = {
  auth: {
    title: "Sign-in required",
    message: "Agent Manager did not accept this session for this data. Open Mizar from the Vrooli console or sign in again, then retry.",
    retryable: true,
  },
  forbidden: {
    title: "Not permitted",
    message: "Your identity is not allowed to read this. Ask an operator to grant access.",
    retryable: false,
  },
  unreachable: {
    title: "Agent Manager is unreachable",
    message: "The API is not responding. Check that the agent-manager scenario is running, then retry.",
    retryable: true,
  },
  misrouted: {
    title: "Unexpected response",
    message: "The request reached a web page instead of the API. The UI may be newer than the running API; restart agent-manager, then retry.",
    retryable: true,
  },
  not_found: {
    title: "Not found",
    message: "This item no longer exists. It may have been deleted.",
    retryable: false,
  },
  timeout: {
    title: "Request timed out",
    message: "Agent Manager took too long to answer. Retry in a moment.",
    retryable: true,
  },
  unimplemented: {
    title: "Not available yet",
    message: "The running Agent Manager API does not serve this yet. It appears after agent-manager is updated and restarted.",
    retryable: false,
  },
};

function errorText(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  return "";
}

function statusCode(text: string): number | undefined {
  const match = /(?:HTTP|Request failed:|status)\s*(\d{3})\b/i.exec(text);
  return match?.[1] ? Number(match[1]) : undefined;
}

function classify(text: string): ErrorKind {
  const lower = text.toLowerCase();
  const code = /^\[([a-z_]+)\]/.exec(lower)?.[1];
  const http = statusCode(text);
  if (code === "unauthenticated" || http === 401) return "auth";
  if (code === "permission_denied" || http === 403) return "forbidden";
  if (lower.includes("unsupported content type text/html") || lower.includes("unexpected token '<'")) return "misrouted";
  if (code === "unavailable" || http === 502 || http === 503 || lower.includes("failed to fetch") || lower.includes("networkerror") || lower.includes("load failed")) return "unreachable";
  if (code === "deadline_exceeded" || http === 504 || lower.includes("timed out") || lower.includes("timeout")) return "timeout";
  if (code === "not_found" || http === 404) return "not_found";
  if (code === "unimplemented" || http === 501) return "unimplemented";
  if (code === "invalid_argument" || code === "failed_precondition" || http === 400 || http === 409 || http === 422) return "rejected";
  return "unknown";
}

/** Strips a Connect "[code]" prefix so a server's own sentence can be shown as-is. */
function serverSentence(text: string): string {
  return text.replace(/^\[[a-z_]+\]\s*/i, "").trim();
}

export function presentError(error: unknown): ErrorPresentation {
  const detail = errorText(error);
  const kind = classify(detail);
  if (kind === "rejected") {
    return { kind, title: "Request was rejected", message: serverSentence(detail) || "Agent Manager refused this request.", detail, retryable: false };
  }
  if (kind === "unknown") {
    const sentence = serverSentence(detail);
    // A short human sentence from the server is more useful than a generic line.
    const readable = sentence.length > 0 && sentence.length <= 160 && !/^(HTTP|Request failed)/i.test(sentence) && !sentence.includes("\n");
    return { kind, title: "Something went wrong", message: readable ? sentence : "The request could not be completed.", detail, retryable: true };
  }
  return { kind, ...COPY[kind], detail };
}
