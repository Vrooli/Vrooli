import type { ApprovalDecision, ApprovalSource } from "../../api/approvals";

/**
 * Small pure helpers shared by the approvals list, detail, banner, and tests.
 * Kept free of React so they test without a renderer.
 */

/** Urgency ordering for the list — higher sorts first. */
const URGENCY_RANK: Record<string, number> = {
  critical: 4,
  high: 3,
  normal: 2,
  low: 1,
  "": 0,
};

export function urgencyRank(urgency: string): number {
  return URGENCY_RANK[(urgency || "").trim()] ?? 0;
}

/**
 * Sort decisions for the list: most urgent first, then soonest deadline, then
 * most recently raised. A deadline-less decision sorts after one with a
 * deadline at the same urgency (it is less time-pressured).
 */
export function sortDecisions(decisions: ApprovalDecision[]): ApprovalDecision[] {
  return [...decisions].sort((a, b) => {
    const ur = urgencyRank(b.urgency) - urgencyRank(a.urgency);
    if (ur !== 0) return ur;
    const ad = a.deadline ? Date.parse(a.deadline) : Number.POSITIVE_INFINITY;
    const bd = b.deadline ? Date.parse(b.deadline) : Number.POSITIVE_INFINITY;
    if (ad !== bd) return ad - bd;
    const ac = a.createdAt ? Date.parse(a.createdAt) : 0;
    const bc = b.createdAt ? Date.parse(b.createdAt) : 0;
    return bc - ac;
  });
}

/**
 * Human relative time like "in 2h", "in 40m", "6m ago", "just now". Pure and
 * deterministic given `now` so tests pin it.
 */
export function relativeTime(iso: string, now: number = Date.now()): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "";
  const deltaMs = t - now;
  const future = deltaMs >= 0;
  const abs = Math.abs(deltaMs);
  const mins = Math.round(abs / 60000);
  let body: string;
  if (mins < 1) return "just now";
  if (mins < 60) body = `${String(mins)}m`;
  else if (mins < 60 * 24) {
    const h = Math.floor(mins / 60);
    const m = mins % 60;
    body = m ? `${String(h)}h ${String(m)}m` : `${String(h)}h`;
  } else {
    const d = Math.floor(mins / (60 * 24));
    body = `${String(d)}d`;
  }
  return future ? `in ${body}` : `${body} ago`;
}

/** The option a producer recommends, resolved to its label. */
export function recommendedOption(decision: ApprovalDecision): { key: string; label: string } | null {
  const opt = decision.options.find((o) => o.key === decision.recommended);
  if (!opt) return null;
  return { key: opt.key, label: opt.label };
}

/**
 * The option key a list-row "approve" gesture should submit. Prefer the
 * producer's recommended option, but ONLY when it is actually one of the
 * decision's options — a producer that sent an invalid `recommended` must not
 * cause the swipe to submit the wrong (first) option. Returns null when there
 * is nothing safe to submit, so the caller opens the detail instead.
 */
export function approveOptionKey(decision: ApprovalDecision): string | null {
  const rec = decision.options.find((o) => o.key === decision.recommended);
  if (rec) return rec.key;
  // No valid recommendation: do NOT guess options[0] for an approve gesture.
  return null;
}

/**
 * Guard a context_url before it is rendered as a link (defense-in-depth against
 * a javascript:/data: payload smuggled into Facts.context_url). Only absolute
 * http(s) URLs and root-relative paths ("/...") are allowed; anything else
 * (including protocol-relative "//host", "javascript:", "data:", "vbscript:")
 * is rejected, and the caller renders the value as inert text instead of a link.
 */
export function safeContextUrl(raw: string): string | null {
  const url = (raw || "").trim();
  if (!url) return null;
  // Root-relative, but NOT protocol-relative "//evil.com".
  if (url.startsWith("/") && !url.startsWith("//")) return url;
  try {
    const parsed = new URL(url);
    if (parsed.protocol === "http:" || parsed.protocol === "https:") return url;
  } catch {
    return null;
  }
  return null;
}

/** Options that are not the recommended one, in declared order. */
export function secondaryOptions(decision: ApprovalDecision) {
  return decision.options.filter((o) => o.key !== decision.recommended);
}

/** A short label for a source, for chips/filters. */
export function sourceLabel(source: ApprovalSource): string {
  switch (source) {
    case "agent-manager":
      return "agent-manager";
    case "deployment-manager":
      return "deployment-manager";
    case "swarm-manager":
      return "swarm-manager";
    case "vrooli-autoheal":
      return "vrooli-autoheal";
    case "notification-hub":
      return "notification-hub";
    default:
      return "system";
  }
}

/**
 * The semantic color token family for a source's accent (no invented colors;
 * these are the ratified reuse of existing wc-* / design tokens — see mockups
 * README). Returned as a Tailwind-ready token name the components interpolate.
 */
export function sourceAccent(source: ApprovalSource): { text: string; bg: string; dot: string } {
  switch (source) {
    case "deployment-manager":
      return { text: "text-wc-warning", bg: "bg-wc-warning/15", dot: "bg-wc-warning" };
    case "swarm-manager":
      // --provenance-sample is already defined in design-tokens.css.
      return { text: "text-[color:var(--provenance-sample)]", bg: "bg-[color:var(--provenance-sample)]/15", dot: "bg-[color:var(--provenance-sample)]" };
    case "vrooli-autoheal":
      return { text: "text-wc-success", bg: "bg-wc-success/15", dot: "bg-wc-success" };
    case "agent-manager":
    case "notification-hub":
    default:
      return { text: "text-wc-accent", bg: "bg-wc-accent/15", dot: "bg-wc-accent" };
  }
}

/** Whether the decision can still be acted on (pending and not past deadline). */
export function isActionable(decision: ApprovalDecision): boolean {
  return decision.state === "pending";
}
