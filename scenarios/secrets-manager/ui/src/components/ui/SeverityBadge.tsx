import { severityAccent } from "../../lib/constants";

export const SeverityBadge = ({ severity }: { severity: string }) => {
  const key = severityAccent[severity]
    ? severity
    : severity === "healthy"
    ? "low"
    : severity === "degraded"
    ? "high"
    : severity === "invalid"
    ? "medium"
    : severity;

  return (
    <span
      className={`rounded-full border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${
        severityAccent[key] ?? severityAccent.low
      }`}
    >
      {severity}
    </span>
  );
};
