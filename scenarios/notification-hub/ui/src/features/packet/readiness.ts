import { strings } from "../../consts/strings";

/** The grades the Hub assigns; empty means it never assessed the packet. */
export type ReadinessGrade = "ready" | "needs_work" | "refused";

export interface ReadinessFinding {
  code: string;
  severity: string;
  field: string;
  owner: string;
  message: string;
}

export interface ReadinessReport {
  grade: ReadinessGrade | "";
  score: number | null;
  findings: ReadinessFinding[];
}

export const READINESS_GRADE = {
  ready: "ready",
  needs_work: "needs_work",
  refused: "refused",
} as const;

/** The word for each grade. A grade is a status, never colour alone. */
export const READINESS_LABEL = {
  ready: strings.asks.row.readiness.ready,
  needs_work: strings.asks.row.readiness.needsWork,
  refused: strings.asks.row.readiness.returned,
} as const satisfies Record<ReadinessGrade, string>;

/** The chip's tone: ready is calm, needs work is attention, returned is a stop. */
export const READINESS_TONE = {
  ready: "success",
  needs_work: "warning",
  refused: "danger",
} as const satisfies Record<ReadinessGrade, "success" | "warning" | "danger">;

export function readinessGrade(value: string | undefined | null): ReadinessGrade | "" {
  return value === "ready" || value === "needs_work" || value === "refused" ? value : "";
}

const record = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const text = (...values: unknown[]): string => {
  for (const value of values) if (typeof value === "string" && value !== "") return value;
  return "";
};

function findingOf(value: unknown): ReadinessFinding | null {
  if (!record(value)) return null;
  const message = text(value.message, value.detail, value.summary);
  const code = text(value.code, value.rule);
  const severity = text(value.severity, value.level);
  if (!message && !code) return null;
  return {
    code,
    severity,
    field: text(value.field, value.path),
    owner: text(value.owner, value.source),
    message,
  };
}

/**
 * Read the Hub's readiness report. The field is JSON that may grow; a report
 * that does not parse is simply not shown, and an unknown finding shape is
 * dropped rather than rendered as raw JSON.
 */
export function parseQualityReport(raw: string | undefined): ReadinessReport | null {
  if (!raw) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!record(parsed)) return null;
  const findings = Array.isArray(parsed.findings)
    ? parsed.findings.flatMap((value): ReadinessFinding[] => {
        const finding = findingOf(value);
        return finding ? [finding] : [];
      })
    : [];
  const grade = readinessGrade(text(parsed.grade));
  const score = typeof parsed.score === "number" && Number.isFinite(parsed.score) ? parsed.score : null;
  if (!grade && findings.length === 0) return null;
  return { grade, score, findings };
}

/** The finding severity in the reader's words; an unknown token reads as advice. */
export const SEVERITY_LABEL = {
  blocker: strings.packet.readiness.severity.blocker,
  warning: strings.packet.readiness.severity.warning,
  advisory: strings.packet.readiness.severity.advisory,
} as const;

export function severityOf(value: string): keyof typeof SEVERITY_LABEL {
  if (value === "blocking" || value === "blocker" || value === "error" || value === "critical") return "blocker";
  if (value === "warning" || value === "warn") return "warning";
  // A missing or future token is not silently the weakest tier.
  if (value === "advice" || value === "advisory" || value === "info" || value === "") return "advisory";
  return "warning";
}

/** "Same cause", "Same subject", "Similar wording 0.72", "Semantically close 0.74". */
export const RELATED_LABEL = {
  family: strings.packet.related.basis.family,
  subject: strings.packet.related.basis.subject,
  lexical: strings.packet.related.basis.lexical,
  semantic: strings.packet.related.basis.semantic,
  other: strings.packet.related.basis.other,
} as const;

export function relatedLabelKey(basis: string): (typeof RELATED_LABEL)[keyof typeof RELATED_LABEL] {
  if (basis === "family" || basis === "subject" || basis === "semantic") return RELATED_LABEL[basis];
  if (basis === "lexical") return RELATED_LABEL.lexical;
  return RELATED_LABEL.other;
}

export function scoreText(score: number): string {
  // A missing score is not "0.00": the label only shows a number it has.
  return Number.isFinite(score) && score > 0 ? score.toFixed(2) : "";
}

export const FOLLOWTHROUGH_STATE_LABEL = {
  queued: strings.packet.followthrough.state.queued,
  running: strings.packet.followthrough.state.running,
  done: strings.packet.followthrough.state.done,
  stalled: strings.packet.followthrough.state.stalled,
  failed: strings.packet.followthrough.state.failed,
} as const;

export type FollowthroughState = keyof typeof FOLLOWTHROUGH_STATE_LABEL;

export function followthroughState(value: string): FollowthroughState | null {
  return Object.prototype.hasOwnProperty.call(FOLLOWTHROUGH_STATE_LABEL, value) ? (value as FollowthroughState) : null;
}

export const FOLLOWTHROUGH_TONE = {
  queued: "neutral",
  running: "info",
  done: "success",
  stalled: "warning",
  failed: "danger",
} as const satisfies Record<FollowthroughState, "neutral" | "info" | "success" | "warning" | "danger">;

/** A bounded run says how long it may take, in the largest whole unit that reads cleanly. */
export type BoundParts = {
  key: typeof strings.packet.followthrough.boundSeconds | typeof strings.packet.followthrough.boundMinutes | typeof strings.packet.followthrough.boundMinutesSeconds;
  count?: number;
  minutes?: number;
  seconds?: number;
};

export function boundParts(seconds: number): BoundParts | null {
  if (!Number.isFinite(seconds) || seconds <= 0) return null;
  const whole = Math.round(seconds);
  if (whole < 60) return { key: strings.packet.followthrough.boundSeconds, count: whole };
  if (whole % 60 === 0) return { key: strings.packet.followthrough.boundMinutes, count: whole / 60 };
  return {
    key: strings.packet.followthrough.boundMinutesSeconds,
    minutes: Math.floor(whole / 60),
    seconds: whole % 60,
  };
}
