export const severityAccent: Record<string, string> = {
  critical: "border-red-500/60 bg-red-500/10 text-red-200",
  high: "border-amber-400/60 bg-amber-400/10 text-amber-100",
  medium: "border-yellow-300/40 bg-yellow-300/5 text-yellow-100",
  low: "border-emerald-400/50 bg-emerald-400/5 text-emerald-100"
};

export type Intent = "good" | "warn" | "danger" | "info";

export const intentAccent: Record<Intent, string> = {
  good: "border-emerald-500/30 bg-emerald-500/5 text-emerald-100",
  warn: "border-amber-400/40 bg-amber-400/10 text-amber-100",
  danger: "border-red-500/40 bg-red-500/10 text-red-100",
  info: "border-cyan-400/40 bg-cyan-400/10 text-cyan-100"
};

export const classificationTone: Record<string, string> = {
  infrastructure: "bg-sky-400/15 text-sky-100 border-sky-500/40",
  service: "bg-purple-400/10 text-purple-100 border-purple-500/40",
  user: "bg-amber-400/10 text-amber-100 border-amber-500/40"
};
